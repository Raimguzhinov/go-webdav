package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/emersion/go-ical"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/joho/godotenv"

	"github.com/Raimguzhinov/go-webdav"
	"github.com/Raimguzhinov/go-webdav/caldav"
)

// transport is an http.RoundTripper that keeps track of the in-flight
// request and implements hooks to report HTTP tracing events.
type transport struct {
	current *http.Request
}

// RoundTrip wraps http.DefaultTransport.RoundTrip to keep track
// of the current request.
func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.current = req
	// fmt.Printf("\n\nRequest:\n%v\n", req)
	resp, err := http.DefaultTransport.RoundTrip(req)
	return resp, err
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println(
			"No .env file found. Please create it with CALDAV_ROOT, CALDAV_USER, CALDAV_PASSWORD environment variables",
		)
		os.Exit(1)
	}
}

func main() {
	root, exists := os.LookupEnv("CALDAV_ROOT")
	if !exists {
		fmt.Println(
			"Please set CALDAV_ROOT environment variable (CALDAV_ROOT=http://caldav.exapmle.com)",
		)
		os.Exit(1)
	}
	user, exists := os.LookupEnv("CALDAV_USER")
	if !exists {
		fmt.Println(
			"Please set CALDAV_USER environment variable",
		)
		os.Exit(1)
	}
	password, exists := os.LookupEnv("CALDAV_PASSWORD")
	if !exists {
		fmt.Println(
			"Please set CALDAV_PASSWORD environment variable",
		)
		os.Exit(1)
	}

	t := &transport{}

	client := &http.Client{Transport: t}

	baHttpClient := webdav.HTTPClientWithBasicAuth(
		client,
		user,
		password,
	)

	caldavClient, err := caldav.NewClient(baHttpClient, root)
	if err != nil {
		log.Fatal(err)
	}
	principal, err := caldavClient.FindCurrentUserPrincipal(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n\nResponse:\n%v\n", principal)
	homeSet, err := caldavClient.FindCalendarHomeSet(context.Background(), principal)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n\nResponse:\n%v\n", homeSet)
	calendars, err := caldavClient.FindCalendars(context.Background(), homeSet)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n\nResponse:\n")
	for i, calendar := range calendars {
		fmt.Printf("cal %d: %s %s\n", i, calendar.Name, calendar.Path)
	}

	if len(calendars) > 0 {

		id, err := uuid.GenerateUUID()
		if err != nil {
			log.Fatalf("could not generate UUID: %v", err)
		}
		id = strings.ToUpper(id)
		_ = id

		// co, err := caldavClient.PutCalendarObject(context.Background(), dst, cal)
		// if err != nil {
		// 	log.Fatalf("could not put calendar event to %q: %v", dst, err)
		// }
		//
		// fmt.Println(co)

		query := &caldav.CalendarQuery{
			CompRequest: caldav.CalendarCompRequest{
				Name:  "VCALENDAR",
				Props: []string{"VERSION"},
				Comps: []caldav.CalendarCompRequest{{
					Name: "VEVENT",
					Props: []string{
						"SUMMARY",
						"DESCRIPTION",
						"UID",
						"DTSTART",
						"DTEND",
						"DURATION",
					},
				}},
			},
			CompFilter: caldav.CompFilter{
				Name: "VCALENDAR",
				Comps: []caldav.CompFilter{{
					Name:  "VEVENT",
					Start: time.Now().Add(-92 * time.Hour),
					End:   time.Now().Add(24 * time.Hour),
				}},
			},
		}

		resp, err := caldavClient.QueryCalendar(
			context.Background(),
			calendars[0].Path,
			query,
		)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("\n\nResponse:\n")
		for i, event := range resp {
			fmt.Printf("ics %d: %s\n", i, event.Path)
		}

		dst, err := url.JoinPath(root, resp[0].Path)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("\n\nEvents:\n")

		for _, icsEvent := range resp {
			req, err := http.NewRequestWithContext(
				context.Background(),
				http.MethodGet,
				dst,
				nil,
			)
			if err != nil {
				log.Fatalf("crafting ics request: %v", err)
			}
			req.SetBasicAuth(user, password)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Fatalf("performing ics request: %v", err)
			}

			events, err := decodeEvents(resp.Body)
			if err != nil {
				log.Fatalf("decoding ics events: %v", err)
			}

			for _, e := range events {
				redacted := redactComponent(e.Component)
				redacted.Props.Set(&ical.Prop{Name: "SUMMARY", Value: "Test"})
				dtstrt, _ := redacted.Props.DateTime("DTSTART", time.Local)
				dtend, _ := redacted.Props.DateTime("DTEND", time.Local)
				dstmp, _ := redacted.Props.DateTime("DTSTAMP", time.Local)

				fmt.Println(icsEvent)
				fmt.Println("DTSTART:", dtstrt)
				fmt.Println("DTEND:", dtend)
				fmt.Println("DTSTAMP:", dstmp)
				fmt.Printf("\n")
			}
		}
	} else {
		fmt.Println("Calendars not found")
	}
}

func decodeEvents(r io.ReadCloser) (events []ical.Event, _ error) {
	dec := ical.NewDecoder(r)
	defer r.Close()

	for {
		cal, err := dec.Decode()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		events = append(events, cal.Events()...)
	}
	return
}

func redactComponent(e *ical.Component) *ical.Component {
	redactedProps := make(ical.Props)

	for k, props := range e.Props {
		mustRedact, ok := REDACT[k]
		if !ok {
			uid, _ := e.Props.Text(ical.PropUID)
			log.Println("redacted unknown property", k, uid)
			continue
		}

		if mustRedact {
			continue
		}

		if k == ical.PropUID {
			for _, p := range props {
				if strings.Contains(p.Value, "@") {
					continue // skip non-UUID
				}
			}
		}
		redactedProps[k] = props
	}

	var redactedChildren []*ical.Component
	for _, child := range e.Children {
		redactedChildren = append(redactedChildren, redactComponent(child))
	}

	return &ical.Component{
		Name:     e.Name,
		Props:    redactedProps,
		Children: redactedChildren,
	}
}

var REDACT = map[string]bool{
	// RFC5545
	"CALSCALE":         false,
	"METHOD":           false,
	"PRODID":           true,
	"VERSION":          false,
	"ATTACH":           true,
	"CATEGORIES":       true,
	"CLASS":            false,
	"COMMENT":          true,
	"DESCRIPTION":      true,
	"GEO":              true,
	"LOCATION":         true,
	"PERCENT-COMPLETE": true,
	"PRIORITY":         false,
	"RESOURCES":        true,
	"STATUS":           false,
	"SUMMARY":          true,
	"COMPLETED":        false,
	"DTEND":            false,
	"DUE":              false,
	"DTSTART":          false,
	"DURATION":         false,
	"FREEBUSY":         false,
	"TRANSP":           false,
	"TZID":             false,
	"TZNAME":           false,
	"TZOFFSETFROM":     false,
	"TZOFFSETTO":       false,
	"TZURL":            false,
	"ATTENDEE":         true,
	"CONTACT":          true,
	"ORGANIZER":        true,
	"RECURRENCE-ID":    false,
	"RELATED-TO":       true,
	"URL":              true,
	"UID":              false,
	"EXDATE":           false,
	"RDATE":            false,
	"RRULE":            false,
	"ACTION":           false,
	"REPEAT":           false,
	"TRIGGER":          false,
	"CREATED":          false,
	"DTSTAMP":          false,
	"LAST-MODIFIED":    false,
	"SEQUENCE":         false,
	"REQUEST-STATUS":   true,

	//
	"ACKNOWLEDGED": false,

	// Non-RFC
	"X-MOZ-LASTACK":    false,
	"X-MOZ-GENERATION": false,
}
