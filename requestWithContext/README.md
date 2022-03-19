`data.go` use `*http.Request` as a key of global request variables. It's a bad implementation because there is a chance to lead to memory leaks.
`main.go` show that situation. Every request to server allocates about 1KB memory that GC can not recover.
These codes explain well why we should not use https://github.com/gorilla/context which has been archived now.


```
2022/03/20 02:09:17 ------start------
2022/03/20 02:09:17 Alloc:153128, Sys: 8735760, NumGC:0
2022/03/20 02:09:18 Alloc:1492768, Sys: 14109456, NumGC:13
2022/03/20 02:09:19 Alloc:1871352, Sys: 14109456, NumGC:25
2022/03/20 02:09:20 Alloc:3167400, Sys: 14109456, NumGC:38
2022/03/20 02:09:21 Alloc:932416, Sys: 14109456, NumGC:52
2022/03/20 02:09:22 Alloc:2049536, Sys: 14109456, NumGC:65
2022/03/20 02:09:23 Alloc:3252496, Sys: 14109456, NumGC:78
2022/03/20 02:09:24 Alloc:3275920, Sys: 14109456, NumGC:91
2022/03/20 02:09:25 Alloc:1297616, Sys: 14109456, NumGC:105
2022/03/20 02:09:26 Alloc:3090416, Sys: 14109456, NumGC:118
2022/03/20 02:09:27 Alloc:1222152, Sys: 14109456, NumGC:132
2022/03/20 02:09:27 http: Server closed
2022/03/20 02:09:27 ------end------
2022/03/20 02:09:27 Alloc:3034584, Sys: 14109456, NumGC:138
2022/03/20 02:09:27 ------GC------
2022/03/20 02:09:27 Alloc:253272, Sys: 14109456, NumGC:139
```

with `r = r.WithContext(context.Background())`

```
2022/03/20 02:08:26 ------start------
2022/03/20 02:08:26 Alloc:153688, Sys: 8735760, NumGC:0
2022/03/20 02:08:27 Alloc:10225264, Sys: 22498064, NumGC:11
2022/03/20 02:08:28 Alloc:22031192, Sys: 30886672, NumGC:16
2022/03/20 02:08:29 Alloc:37135296, Sys: 48384784, NumGC:19
2022/03/20 02:08:30 Alloc:33454688, Sys: 65489680, NumGC:22
2022/03/20 02:08:31 Alloc:60020816, Sys: 74009360, NumGC:23
2022/03/20 02:08:32 Alloc:79574832, Sys: 93640480, NumGC:24
2022/03/20 02:08:33 Alloc:96369640, Sys: 110745376, NumGC:25
2022/03/20 02:08:34 Alloc:107231496, Sys: 123508512, NumGC:26
2022/03/20 02:08:35 Alloc:109124984, Sys: 136419104, NumGC:27
2022/03/20 02:08:36 Alloc:102342848, Sys: 160146224, NumGC:28
2022/03/20 02:08:37 Alloc:154539776, Sys: 173196080, NumGC:28
2022/03/20 02:08:37 http: Server closed
2022/03/20 02:08:37 ------end------
2022/03/20 02:08:37 Alloc:162086856, Sys: 181584688, NumGC:28
2022/03/20 02:08:38 ------GC------
2022/03/20 02:08:38 Alloc:94844056, Sys: 181584688, NumGC:30
```

Instead, we should use http.Request.Context to store UserID like this:

```go
// set
r = r.WithContext(context.WithValue(r.Context(), "userID", 2))
// get
r.Context().Value("userID")
```
