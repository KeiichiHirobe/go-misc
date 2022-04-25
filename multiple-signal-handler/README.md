## Output
```
2022/04/25 17:44:39 9 goroutine catch signal
2022/04/25 17:44:39 7 goroutine catch signal
2022/04/25 17:44:39 0 goroutine catch signal
2022/04/25 17:44:39 8 goroutine catch signal
2022/04/25 17:44:39 1 goroutine catch signal
2022/04/25 17:44:39 3 goroutine catch signal
2022/04/25 17:44:39 2 goroutine catch signal
2022/04/25 17:44:39 6 goroutine catch signal
2022/04/25 17:44:39 4 goroutine catch signal
2022/04/25 17:44:42 5 goroutine catch signal
2022/04/25 17:44:42 all of task get signal
```

## Related golang source code
https://github.com/golang/go/blob/96c8cc7fea94dca8c9e23d9653157e960f2ff472/src/os/signal/signal.go#L149-L153
https://github.com/golang/go/blob/96c8cc7fea94dca8c9e23d9653157e960f2ff472/src/os/signal/signal_unix.go#L19-L28
https://github.com/golang/go/blob/96c8cc7fea94dca8c9e23d9653157e960f2ff472/src/os/signal/signal.go#L232-L260

runtime implements `func signal_recv() uint32` at
https://github.com/golang/go/blob/96c8cc7fea94dca8c9e23d9653157e960f2ff472/src/runtime/sigqueue.go#L126-L168
