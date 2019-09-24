# go-timewheel

golang timewheeo lib, similar to goalng std timer

## Usage

init timewheel

```
tw, err := NewTimeWheel(1 * time.Second, 360)
if err != nil {
    panic(err)
}

tw.Start()
tw.Stop()
```

add delay task

```
task, err := tw.Add(5 * time.Second, func(){})
```

remove delay task

```
tw.Remove(task)
```

similar to time.Sleep

```
tw.Sleep(5 * time.Second)
```

similar to time.NewTimer

```
timer :=tw.NewTimer(5 * time.Second)
<- timer.C
timer.Reset(1 * time.Second)
timer.Stop()
```

similar to time.NewTicker

```
timer :=tw.NewTicker(5 * time.Second)
<- timer.C
timer.Stop()
```

## benchmark test

[example/main.go](example/main.go)
