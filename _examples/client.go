package client

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Raimguzhinov/go-webdav"
	"github.com/Raimguzhinov/go-webdav/caldav"
	"github.com/emersion/go-ical"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/joho/godotenv"
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

func RunClient() {
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
	log.Println(principal + "\n")
	homeSet, err := caldavClient.FindCalendarHomeSet(context.Background(), principal)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(homeSet + "\n")
	calendars, err := caldavClient.FindCalendars(context.Background(), homeSet)
	if err != nil {
		log.Fatal(err)
	}
	for i, calendar := range calendars {
		fmt.Printf("cal %d: %s %s\n", i, calendar.Name, calendar.Path)
	}

	if len(calendars) > 0 {

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
						"LAST-MODIFIED",
						"X-PROTEI-SENDERID",
					},
				}},
			},
			CompFilter: caldav.CompFilter{
				Name: "VCALENDAR",
				Comps: []caldav.CompFilter{{
					Name: "VEVENT",
					// Start: time.Now().Add(-92 * time.Hour),
					// End:   time.Now().Add(24 * time.Hour),
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

		for _, calendarObject := range resp {
			for _, e := range calendarObject.Data.Events() {
				// fmt.Printf("\n\nEVENT PROPS:\n%v\n", e.Props)
				location := time.Local
				eventClass := GetPropertyValue(e.Props.Get("CLASS"))
				eventId := GetPropertyValue(e.Props.Get("UID"))
				eventName, _ := e.Props.Text("SUMMARY")
				eventDescription, _ := e.Props.Text("DESCRIPTION")
				eventUrl := GetPropertyValue(e.Props.Get("URL"))
				eventSenderId := GetPropertyValue(e.Props.Get("X-PROTEI-SENDERID"))
				sequence := GetPropertyValue(e.Props.Get("SEQUENCE"))
				transp := GetPropertyValue(e.Props.Get("TRANSP"))
				status, _ := e.Status()

				createdTime, err := e.Props.DateTime("CREATED", location)
				if err != nil {
					log.Fatal(err, "Can't parse CREATED for event "+eventName)
				}
				startTime, err := e.Props.DateTime("DTSTART", location)
				if err != nil {
					log.Fatal(err, "Can't parse DTSTART for event "+eventName)
				}
				endTime, err := e.Props.DateTime("DTEND", location)
				if err != nil {
					log.Fatal(err, "Can't parse DTEND for event "+eventName)
				}
				dtstamp, err := e.Props.DateTime("DTSTAMP", location)
				if err != nil {
					log.Fatal(err, "Can't parse DTSTAMP for event "+eventName)
				}
				lastModifiedTime, err := e.Props.DateTime("LAST-MODIFIED", location)
				if err != nil {
					log.Fatal(err, "Can't parse LAST-MODIFIED for event "+eventName)
				}

				fmt.Printf(
					"\nCLASS: %s\nSTATUS: %s\nUID: %s\nSUMMARY: %s\nDESCRIPTION: %s\nURL: %s\nCREATED: %s\nDTSTART: %s\nDTEND: %s\nDTSTAMP %s\nLAST-MODIFIED: %s\nSEQUENCE: %s\nTRANSP: %s\nX-PROTEI-SENDERID: %s\n",
					eventClass,
					status,
					eventId,
					eventName,
					eventDescription,
					eventUrl,
					createdTime.Local(),
					startTime,
					endTime,
					dtstamp.Local(),
					lastModifiedTime.Local(),
					sequence,
					transp,
					eventSenderId,
				)
			}
		}

	} else {
		fmt.Println("Calendars not found")
	}

	id, err := uuid.GenerateUUID()
	if err != nil {
		log.Fatalf("could not generate UUID: %v", err)
	}

	alarm := ical.NewComponent(ical.CompAlarm)
	alarm.Props.SetText(ical.PropAction, "DISPLAY")
	trigger := ical.NewProp(ical.PropTrigger)
	trigger.SetDuration(-58 * time.Minute)
	alarm.Props.Set(trigger)

	eventSummary := "protei event tzid"
	event := ical.NewEvent()
	event.Name = ical.CompEvent
	event.Props.SetDateTime(ical.PropCreated, time.Now().UTC())
	event.Props.SetDateTime(ical.PropDateTimeStamp, time.Now().UTC())
	event.Props.SetDateTime(ical.PropLastModified, time.Now().UTC())
	event.Props.SetText(ical.PropSequence, "1")
	event.Props.SetText(ical.PropUID, id)
	event.Props.SetDateTime(ical.PropDateTimeStart, time.Now().Local().Add(1*time.Hour))
	event.Props.SetDateTime(ical.PropDateTimeEnd, time.Now().Local().Add(2*time.Hour))
	event.SetStatus(ical.EventConfirmed)
	event.Props.SetText(ical.PropSummary, eventSummary)
	event.Props.SetText(ical.PropTransparency, "OPAQUE")
	event.Props.SetText(ical.PropClass, "PUBLIC")

	sender := ical.NewProp("X-PROTEI-SENDERID")
	sender.SetText("12345")
	event.Props.Set(sender)

	event.Children = []*ical.Component{
		alarm,
	}

	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropProductID, "-//Raimguzhinov//go-caldav 1.0//EN")
	cal.Props.SetText(ical.PropCalendarScale, "GREGORIAN")

	// standard := ical.NewComponent(ical.CompTimezoneStandard)
	// tzid, _ := time.Now().Zone()
	// standard.Props.SetText(ical.PropTimezoneName, tzid)
	// standard.Props.SetDateTime(ical.PropDateTimeStart, time.Now().Local())
	// standard.Props.SetText(ical.PropTimezoneOffsetTo, "+0700")
	// standard.Props.SetText(ical.PropTimezoneOffsetFrom, "+0700")
	//
	// timezone := ical.NewComponent(ical.CompTimezone)
	// timezone.Props.SetText(ical.PropTimezoneID, "Asia/Novosibirsk")
	// timezone.Children = []*ical.Component{
	// 	standard,
	// }

	cal.Children = []*ical.Component{
		event.Component,
	}

	// err = caldavClient.RemoveAll(
	// 	context.Background(),
	// 	calendars[0].Path+"aee29dd1-3c2b-20bb-df22-1b401ada9688.ics",
	// )
	// if err != nil {
	// 	log.Fatalf("could not remove calendar object: %v", err)
	// }

	// obj, err := caldavClient.PutCalendarObject(
	// 	context.Background(),
	// 	calendars[0].Path+id+".ics",
	// 	cal,
	// )
	// if err != nil {
	// 	log.Fatalf("could not put calendar object: %v", err)
	// }
	// fmt.Println(obj)
}

func GetPropertyValue(prop *ical.Prop) string {
	if prop == nil {
		return ""
	}
	return prop.Value
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
	"DESCRIPTION":      false,
	"GEO":              true,
	"LOCATION":         true,
	"PERCENT-COMPLETE": true,
	"PRIORITY":         false,
	"RESOURCES":        true,
	"STATUS":           false,
	"SUMMARY":          false,
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
	"X-MOZ-LASTACK":     false,
	"X-MOZ-GENERATION":  false,
	"X-PROTEI-SENDERID": false,
}
