# go-timewheel

golang timewheeo lib, similar to goalng std timer

## Usage

### base method

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
task := tw.Add(5 * time.Second, func(){})
```

remove delay task

```
tw.Remove(task)
```

add cron delay task

```
task := tw.AddCron(5 * time.Second, func(){
    ...
})
```

### similar to std time

similar to time.Sleep

```
tw.Sleep(5 * time.Second)
```

similar to time.After()

```
<- tw.After(5 * time.Second)
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

similar to time.AfterFunc

```
runner :=tw.NewTicker(5 * time.Second, func(){})
<- runner.C
runner.Stop()
```

## benchmark test

[example/main.go](example/main.go)
