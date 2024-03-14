# go-webdav

[![Go Reference](https://pkg.go.dev/badge/github.com/emersion/go-webdav.svg)](https://pkg.go.dev/github.com/emersion/go-webdav)
[![builds.sr.ht status](https://builds.sr.ht/~emersion/go-webdav/commits/master.svg)](https://builds.sr.ht/~emersion/go-webdav/commits/master?)

A Go library for [WebDAV], [CalDAV] and [CardDAV].

## License

MIT

[WebDAV]: https://tools.ietf.org/html/rfc4918
[CalDAV]: https://tools.ietf.org/html/rfc4791
[CardDAV]: https://tools.ietf.org/html/rfc6352

# CalDAV Client Example

In this example, we demonstrate how to use a CalDAV client to interact with a CalDAV server. The following steps are performed:

1. Initialization of an HTTP client with basic authentication.
2. Creation of a CalDAV client and handling.
3. Finding the current user principal.
4. Searching for the calendar home set.
5. Finding a list of calendars with their names and paths.

## Code Example

- Initialization of an HTTP client with basic authentication.
```go
baHttpClient := webdav.HTTPClientWithBasicAuth(
    client,
    user,
    password,
)
```
- Creation of a CalDAV client and handling.
```go
caldavClient, err := caldav.NewClient(baHttpClient, root)
if err != nil {
    log.Fatal(err)
}
```
- Finding the current user principal.
```go
principal, err := caldavClient.FindCurrentUserPrincipal(context.Background())
if err != nil {
    log.Fatal(err)
}
```
- Searching for the calendar home set.
```go
homeSet, err := caldavClient.FindCalendarHomeSet(context.Background(), principal)
if err != nil {
    log.Fatal(err)
}
```
- Finding a list of calendars with their names and paths.
```go
calendars, err := caldavClient.FindCalendars(context.Background(), homeSet)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("\n\nResponse:\n")
for i, calendar := range calendars {
    fmt.Printf("cal %d: %s %s\n", i, calendar.Name, calendar.Path)
}
```
## Installation

Make sure you have the required dependencies installed using go get.

```bash
go get github.com/Raimguzhinov/go-webdav
go get github.com/Raimguzhinov/go-webdav/caldav
```

## Usage

1. Replace client, user, password, and root with your actual values.
2. Run the code and follow the output messages.

This is a simple demonstration of interacting with a CalDAV server using the go-webdav/caldav library in Go!

