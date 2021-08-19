package util
import (
  "os"
  "crypto/md5"
  "encoding/hex"
)

func GetEnv(key string, defaultValue string) string {
    value := os.Getenv(key)
    if len(value) == 0 {
        return defaultValue
    }
    return value
}

func Md5hash(text string) string {
    hasher := md5.New()
    hasher.Write([]byte(text))
    return hex.EncodeToString(hasher.Sum(nil))
}

func ANotInB(a, b []string) (diff []string) {
      m := make(map[string]bool)
      duplicateA := make(map[string]bool)
      for _, item := range b {
              m[item] = true
      }

      for _, item := range a {
        if _, exists := duplicateA[item]; !exists {
              if _, exists := m[item]; !exists {
                      diff = append(diff, item)
                      duplicateA[item] = true
              }
            }
      }
      return
}

// array of string values from orbit.all()
func Aosvfoa(m map[string][]byte) (res []string) {
  for _, v := range m {
          res = append(res, string(v))
  }
  return
}
