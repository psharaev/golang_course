Делаем бенчмарк:
```bash
go test -bench . -benchmem -cpuprofile=cpu.out -memprofile=mem.out -memprofilerate=1 .
```
Смотрим результаты:
```bash
go tool pprof cpu.out
```

Для ускорения работы использовал кодогенерацию (easyjson) для парсинга json. Переделал алгоритм. Избавился от регулярок.
Потребление памяти оптимизировал исходя из предыдущих пунктов.

вывод в `pprof` `list <название_метода>`:
Потребление cpu SlowSearch
```go
Total: 3.66s
ROUTINE ======================== hw3.SlowSearch in /Users/pa.sharaev/Developer/stepik/hw3-bench/common.go
0      1.53s (flat, cum) 41.80% of Total
.          .     16:func SlowSearch(out io.Writer) {
.       10ms     17:   file, err := os.Open(filePath)
.          .     18:   if err != nil {
.          .     19:           panic(err)
.          .     20:   }
.          .     21:
.          .     22:   fileContents, err := ioutil.ReadAll(file)
.          .     23:   if err != nil {
.          .     24:           panic(err)
.          .     25:   }
.          .     26:
.          .     27:   r := regexp.MustCompile("@")
.          .     28:   seenBrowsers := []string{}
.          .     29:   uniqueBrowsers := 0
.          .     30:   foundUsers := ""
.          .     31:
.          .     32:   lines := strings.Split(string(fileContents), "\n")
.          .     33:
.          .     34:   users := make([]map[string]interface{}, 0)
.          .     35:   for _, line := range lines {
.          .     36:           user := make(map[string]interface{})
.          .     37:           // fmt.Printf("%v %v\n", err, line)
.      380ms     38:           err := json.Unmarshal([]byte(line), &user)
.          .     39:           if err != nil {
.          .     40:                   panic(err)
.          .     41:           }
.          .     42:           users = append(users, user)
.          .     43:   }
.          .     44:
.          .     45:   for i, user := range users {
.          .     46:
.          .     47:           isAndroid := false
.          .     48:           isMSIE := false
.          .     49:
.          .     50:           browsers, ok := user["browsers"].([]interface{})
.          .     51:           if !ok {
.          .     52:                   // log.Println("cant cast browsers")
.          .     53:                   continue
.          .     54:           }
.          .     55:
.          .     56:           for _, browserRaw := range browsers {
.          .     57:                   browser, ok := browserRaw.(string)
.          .     58:                   if !ok {
.          .     59:                           // log.Println("cant cast browser to string")
.          .     60:                           continue
.          .     61:                   }
.      580ms     62:                   if ok, err := regexp.MatchString("Android", browser); ok && err == nil {
.          .     63:                           isAndroid = true
.          .     64:                           notSeenBefore := true
.          .     65:                           for _, item := range seenBrowsers {
.          .     66:                                   if item == browser {
.          .     67:                                           notSeenBefore = false
.          .     68:                                   }
.          .     69:                           }
.          .     70:                           if notSeenBefore {
.          .     71:                                   // log.Printf("SLOW New browser: %s, first seen: %s", browser, user["name"])
.          .     72:                                   seenBrowsers = append(seenBrowsers, browser)
.          .     73:                                   uniqueBrowsers++
.          .     74:                           }
.          .     75:                   }
.          .     76:           }
.          .     77:
.          .     78:           for _, browserRaw := range browsers {
.          .     79:                   browser, ok := browserRaw.(string)
.          .     80:                   if !ok {
.          .     81:                           // log.Println("cant cast browser to string")
.          .     82:                           continue
.          .     83:                   }
.      560ms     84:                   if ok, err := regexp.MatchString("MSIE", browser); ok && err == nil {
.          .     85:                           isMSIE = true
.          .     86:                           notSeenBefore := true
.          .     87:                           for _, item := range seenBrowsers {
.          .     88:                                   if item == browser {
.          .     89:                                           notSeenBefore = false
```

Потребление cpu FastSearch
```go
Total: 3.66s
ROUTINE ======================== hw3.FastSearch in /Users/pa.sharaev/Developer/stepik/hw3-bench/fast.go
0      1.21s (flat, cum) 33.06% of Total
.          .     21:func FastSearch(out io.Writer) {
.          .     22:   file, err := os.Open(filePath)
.          .     23:   if err != nil {
.          .     24:           panic(err)
.          .     25:   }
.          .     26:   defer file.Close()
.          .     27:
.          .     28:   i := 0
.          .     29:
.          .     30:   user := User{}
.          .     31:   browsers := map[string]bool{}
.          .     32:
.          .     33:   fmt.Fprintln(out, "found users:")
.          .     34:
.          .     35:   scanner := bufio.NewScanner(file)
.      1.15s     36:   for scanner.Scan() {
.       40ms     37:           if err = user.UnmarshalJSON(scanner.Bytes()); err != nil {
.          .     38:                   panic(err)
.          .     39:           }
.          .     40:
.          .     41:           isAndroid := false
.          .     42:           isMSIE := false
.          .     43:
.          .     44:           for _, browser := range user.Browsers {
.          .     45:                   if strings.Contains(browser, "Android") {
.          .     46:                           isAndroid = true
.          .     47:                   } else if strings.Contains(browser, "MSIE") {
.          .     48:                           isMSIE = true
.          .     49:                   } else {
.          .     50:                           continue
.          .     51:                   }
.          .     52:
.          .     53:                   browsers[browser] = true
.          .     54:           }
.          .     55:
.          .     56:           if isAndroid && isMSIE {
.          .     57:                   email := strings.Replace(user.Email, "@", " [at] ", -1)
.       20ms     58:                   fmt.Fprintln(out, fmt.Sprintf("[%d] %s <%s>", i, user.Name, email))
.          .     59:           }
.          .     60:
.          .     61:           i++
.          .     62:   }
```
