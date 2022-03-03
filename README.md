# go-dispatcher2
The GoLang version of Dispatcher2 - The Data Exchange Middleware

# About

# Deployment
Being a Go application, go-dispatcher2 compiles to a binary and that binary along with the config file is all
you need to run it on your server. You can find bundles for each platform in the
[releases directory](https://github.com/gcinnovate/go-dispatcher2/releases). We recommend running go-dispatcher2
behind a reverse proxy such as nginx that provides HTTPs encryption.

# Configuration
Install go-dispatcher2 source in your workspace with:

```
go get github.com/gcinnovate/go-dispatcher2
```

Build go-dispatcher2 with:

```
go build github.com/gcinnovate/go-dispatcher2
```

This will create a new executable in your current directory `go-dispatcher2`

To run the tests you need to create the test database:

```
$ createdb godispatcher2_test
$ createuser -P -E -s godispatcher2_test (set no password)

To run all of the tests:

```
go test ./... -p=1
```
