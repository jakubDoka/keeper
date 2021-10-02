# keeper

Keeper is simple framework inspired by [nakama](https://github.com/heroiclabs/nakama) that gives lot more freedom, but of course also does not offer that much features (for now). Dependency is kept on minimum (only yaml and postgres driver).

You can use this library in two ways. First the simplest one, implement your server logic trough Modules. Or you can explore the source code and use this all library. I will explain only the first strategy.

## Modules

Only package you really need to import is `core` and call the launch method like so.

```go
package main

import (
    "github.com/jakubDoka/keeper/core"
)

func main() {
    core.Launch()
}
```

No server cant really do anything just yet. We cant even start it actually.

```log
failed to load config, (using default): open config.yaml: The system cannot find the file specified.
<INFO> 2021/09/23 16:07:39 Connecting database...
<ERROR> 2021/09/23 16:07:39
C:/Users/jakub/Documents/programming/golang/src/github.com/jakubDoka/keeper/core/core.go:42
cannot connect to database: pq: password authentication failed for user "postgres"
```

You can get a different error if you don't have Postgres properly installed. You can of course use any driver and any database as keeper does not use any database dependant code. First message is hinting we are missing a config file. Lets generate one and modify it as we need.

```go
    // import "github.com/jakubDoka/keeper/cfg"
    err := kcfg.GenerateConfig()
    if err != nil {
        panic(err)
    }
    // PS: delete the code after first use
```

config.yaml

```yaml
db:
    driver: postgres
    ssl_mode: disable
    cert_file: ""
    key_file: ""
    custom_connection_string: ""
    name: keeper
    user: postgres
    pass: postgres
    host: localhost
    port: 5432
net:
    host: 127.0.0.1
    port: 8080
    cert_file: ""
    key_file: ""
log:
    level: info
    log_to_console: true
    stacktrace_depth:
        error: 1
        warn: 0
        info: 0
        debug: 1
```

Now as we had a problem with database, we have to modify config to make it work. If your database driver does not like the formatting you can use `custom_connection_string` that will be passed directly otherwise leave it empty. Don't forget to create a database with the name you chosen.

```log
<INFO> 2021/09/23 16:39:00 Connecting database...
<WARN> 2021/09/23 16:39:00 Database is running without ssl.
<INFO> 2021/09/23 16:39:00 Initializing router...
<INFO> 2021/09/23 16:39:00 Listening TCP (127.0.0.1:8080)...
<INFO> 2021/09/23 16:39:00 Listening UDP (127.0.0.1:8080)...
<INFO> 2021/09/23 16:39:00 Registered rpc: create-match
<INFO> 2021/09/23 16:39:00 Starting HTTP server (127.0.0.1:8081)...
<WARN> 2021/09/23 16:39:00 Rpc server is running without ssl.
```

Lovely! Server is now running but it still cant really do much. Users can already use create-match rpc but they need session for that. For registering some rpc to obtain session we can use modules. So lets declare module.

```go
package main

import (
    "github.com/jakubDoka/keeper/core"
)

func main() {
    core.Launch(MyFirstModule{})
}

type MyFirstModule struct{}

func (MyFirstModule) Init(i core.InitState) {
    i.Logger.Info("Hello, World!")
}
```

Now if you run this, new log messages will appear:

```log
<INFO> 2021/09/23 16:47:19 Registered rpc: create-match
<INFO> 2021/09/23 16:47:19 Loading MyFirstModule module...
<INFO> 2021/09/23 16:47:19 Hello, World!
<INFO> 2021/09/23 16:47:19 Starting HTTP server (127.0.0.1:8081)...
```

Now lets register that verification rpc that will give session to our user. Keeper already expects you to use email verification so there is api that makes process nicer:

```go
    // inside init function
    i.RegisterEmailLoginHandler(func(s *state.State, email, password, addr string) (*state.User, error) {
        s.Debug("Email login handler called with email %s and password %s from address %s", email, password, addr)
        return state.NewUser(uuid.New(), uuid.New(), 10*time.Second, addr), nil
    })
```

This is minimal example that you should not use. First of all it creates user with ne id and session for free, absolutely ignoring email and password. Duration of session is only 10 seconds which is also very short. Logging password is also best idea. What we want with this is to test our new rpc. Don't forget to verify that keeper logged your rpc registration. Now we have to mock the rpc, (well we don't but still).

...

the README is unfinished
