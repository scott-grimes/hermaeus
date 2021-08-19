package main

import (
    "fmt"
    "log"
    "net/http"
    "net/url"
    "text/template"
    "context"
    "time"
    // "github.com/go-redis/redis/v8"
    "github.com/hibiken/asynq"
    "example.com/m/v2/store"
    "example.com/m/v2/util"
    "example.com/m/v2/worker/tasks"

    // "example.com/m/v2/conf"
)

var s store.Store

type Add struct {
  Address string
  Prefixes string
}

var tmpl = template.Must(template.ParseGlob("form/*"))

func Index(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get("id")
    bannerMessage := s.BannerMessage
    if id == "" {
      tmpl.ExecuteTemplate(w, "Index", struct {
          BannerMessage string
          QueryId string
      }{bannerMessage, id})
    } else {
      ctx, _ := context.WithTimeout(context.Background(), 20*time.Second)
      entry, err := s.Get(id, ctx)
      if err != nil {
        fmt.Println(err)
        http.Redirect(w, r, "error", 302)
      }
      if entry.IpfsRef == "" {
        // todo searching in progress
        // create context. search again after expiry
        task, err := tasks.NewDocFetchTask(id)
        if err != nil {
            fmt.Println(err)
        }
        if s.Config.EnableWorkersForwarding {
          info, err := s.AsyncClient.Enqueue(task, asynq.Unique(time.Minute * 5))
          if err != nil {
              log.Println("could not enqueue task: %v", err)
          }
          ctx, _ := context.WithTimeout(context.Background(), 20*time.Second)
          // todo wait until response recieved or show 'nothing found'
          fmt.Println(info)
          fmt.Println(ctx)
          var exceeded error
          for exceeded != nil {
            exceeded = ctx.Err()
            entry, err = s.Get(id, ctx)
            if err != nil {
              fmt.Println(err)
              http.Redirect(w, r, "error", 302)
            }
            if entry.IpfsRef != "" {
              break;
            }
          }
        }
        // info.State.String() "archived"
        // TaskStateArchived
        // todo check every few times for result. if found display,
        // if at end still nothing display not found
        tmpl.ExecuteTemplate(w, "Index", struct {
            BannerMessage string
            QueryId string
        }{bannerMessage,id})
      } else {
        // found entry in db, display its page
        err := store.VerifyEntry(entry)
        if err != nil {
          fmt.Println("verify failed for entry")
        }
        http.Redirect(w, r, "view?id=" + url.QueryEscape(id), 302)
      }
    }
}

type Results struct  {
  Id string
  IpfsRef string
}

// func All(w http.ResponseWriter, r *http.Request) {
//     selDB, err := store.db.Query("SELECT * FROM Document ORDER BY id DESC")
//     res := []Results{}
//     if err != nil {
//         panic(err.Error())
//     }
//     for selDB.Next() {
//         var id, ipfsRef string
//         err = selDB.Scan(&id, &ipfsRef)
//         if err != nil {
//             panic(err.Error())
//         }
//         print(id, ipfsRef)
//         res = append(res, struct {
//             Id string
//             IpfsRef string
//         }{id, ipfsRef})
//     }
//     tmpl.ExecuteTemplate(w, "All", res)
// }

func Error(w http.ResponseWriter, r *http.Request) {
    tmpl.ExecuteTemplate(w, "Error", nil)
}

func View(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get("id")
    if id == "" {
      http.Redirect(w, r, "index", 302)
    } else {
      ctx, _ := context.WithTimeout(context.Background(), 20*time.Second)
      entry, err := s.Get(id, ctx)
      if err != nil || entry.IpfsRef == "" {
        http.Redirect(w, r, "index?id=" + url.QueryEscape(id), 302)
        return
      }
      tmpl.ExecuteTemplate(w, "View", entry)
    }
}

func New(w http.ResponseWriter, r *http.Request) {
    tmpl.ExecuteTemplate(w, "New", nil)
}

func Edit(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get("id")
    if id == "" {
      http.Redirect(w, r, "index", 302)
    } else {
      ctx, _ := context.WithTimeout(context.Background(), 20*time.Second)
      entry, err := s.Get(id, ctx)
      if err != nil || entry.IpfsRef == "" {
        http.Redirect(w, r, "index?id=" + url.QueryEscape(id), 302)
        return
      }
      tmpl.ExecuteTemplate(w, "Edit", entry)
    }
}

func Insert(w http.ResponseWriter, r *http.Request) {

    if r.Method == "POST" {
        id := r.FormValue("id")
        content := r.FormValue("content")
        var ipfsRef string
        // 10 MB max
        r.ParseMultipartForm(10 << 20)
        file, _, err := r.FormFile("file")
        if err != nil {
            fmt.Println("Error Uploading the File")
            fmt.Println(err)
            http.Redirect(w, r, "error", 302)
            return
        }
        defer file.Close()
        // todo verify document does not already exist
        ctx, _ := context.WithTimeout(context.Background(), 20*time.Second)
        entry, err := s.Get(id, ctx)

        if err != nil {
            fmt.Println("Error Checking Existing Database for File")
            fmt.Println(err)
            http.Redirect(w, r, "error", 302)
            return
        }
        fmt.Println(entry)
        if entry.IpfsRef == "" {

            saveCtx, _ := context.WithTimeout(context.Background(), 20*time.Second)
            var saveErr error
            ipfsRef, saveErr = s.SaveDocument(saveCtx, file)
            if saveErr != nil {
              fmt.Println(err.Error())
              fmt.Println("Error Saving File to IPFS")
              http.Redirect(w, r, "error", 302)
              return
            }
            putCtx, _ := context.WithTimeout(context.Background(), 20*time.Second)
            putErr := s.Put(store.Entry{
              Id: id,
              Content: content,
              IpfsRef: ipfsRef,
              }, putCtx)
            if putErr != nil {
              fmt.Println(putErr.Error())
              fmt.Println("Error Saving File in Database")
              http.Redirect(w, r, "error", 302)
              return
            }
            log.Println("INSERT: Id: " + id + " | IpfsRef: " + ipfsRef)
        }
        http.Redirect(w, r, "index?id=" + url.QueryEscape(id), 302)
    }
}


func Contribute(w http.ResponseWriter, r *http.Request) {

    res := []Add{}

    for storeAddress, prefixes := range s.GetUniqueSubstores() {
      res = append(res, Add{
        Address: storeAddress,
        Prefixes: prefixes,
      })
    }

    tmpl.ExecuteTemplate(w, "Contribute", res)

}


// func Update(w http.ResponseWriter, r *http.Request) {
//     if r.Method == "POST" {
//         id := r.FormValue("id")
//         ipfsRef := r.FormValue("ipfsRef")
//         insForm, err := store.db.Prepare("UPDATE Document SET ipfsRef=? WHERE id=?")
//         if err != nil {
//             panic(err.Error())
//         }
//         insForm.Exec(ipfsRef, id)
//         log.Println("UPDATE: Id: " + id + " | IpfsRef: " + ipfsRef)
//     }
//     http.Redirect(w, r, "/", 301)
// }


// func Delete(w http.ResponseWriter, r *http.Request) {
//     // todo tell to unpin
//     if r.Method == "POST" {
//       id := r.FormValue("id")
//       delForm, err := store.db.Prepare("DELETE FROM Document WHERE id=?")
//       if err != nil {
//           panic(err.Error())
//       }
//       delForm.Exec(id)
//       log.Println("DELETED: " + id)
//     }
//     http.Redirect(w, r, "/", 301)
// }

func Download(w http.ResponseWriter, r *http.Request) {
  id := r.URL.Query().Get("id")
  if id == "" {
    http.NotFound(w,r)
  } else {
    ctx, _ := context.WithTimeout(context.Background(), 20*time.Second)
    entry, err := s.Get(id, ctx)
    if err != nil || entry.IpfsRef == "" {
    http.NotFound(w,r)
    }
    downloadCtx, _ := context.WithTimeout(context.Background(), 20*time.Second)
    file, err := s.GetDocument(downloadCtx, entry.IpfsRef)
    if err != nil || entry.IpfsRef == "" {
      http.NotFound(w,r)
    }
    http.ServeContent(w, r, entry.IpfsRef + ".pdf", time.Now(), file)
  }
}

func Exists(w http.ResponseWriter, r *http.Request) {
  id := r.URL.Query().Get("id")
  if id == "" {
    http.NotFound(w,r)
  } else {
    ctx, _ := context.WithTimeout(context.Background(), 20*time.Second)
    entry, err := s.Get(id, ctx)
    if err != nil || entry.IpfsRef == "" {
      http.NotFound(w,r)
    }
    // http.FOUND
  }
}

func main() {
    port := util.GetEnv("PORT", "8080")
    var initError error

    // if no snapshotting
      // wait for replicated before resource is done

    s, initError = store.NewStore(&store.CreateStoreOptions{
      EnableCache: false,
      ReadOnlyOrbit: true,
      EnableSnapshots: true,
      EnableWorkersForwarding: true,
    })
    if initError != nil {
      panic(initError)
    }


    log.Println("Server started on: http://0.0.0.0:" + port)
    http.HandleFunc("/", Index)
    http.HandleFunc("/view", View)
    http.HandleFunc("/new", New)
    http.HandleFunc("/edit", Edit)
    http.HandleFunc("/insert", Insert)
    http.HandleFunc("/contribute", Contribute)
    // http.HandleFunc("/update", Update)
    // http.HandleFunc("/delete", Delete)
    http.HandleFunc("/error", Error)
    http.HandleFunc("/download", Download)
    // http.HandleFunc("/all", All)
    http.ListenAndServe(":"+port, nil)
}
