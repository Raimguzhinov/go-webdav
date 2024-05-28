package example

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav/caldav"
)

type backend struct {
	calendars []caldav.Calendar
	objectMap map[string][]caldav.CalendarObject
}

func (s *backend) CreateCalendar(ctx context.Context, calendar *caldav.Calendar) error {
	return nil
}

func (s *backend) ListCalendars(ctx context.Context) ([]caldav.Calendar, error) {
	return s.calendars, nil
}

func (s *backend) GetCalendar(ctx context.Context, path string) (*caldav.Calendar, error) {
	for _, cal := range s.calendars {
		if cal.Path == path {
			return &cal, nil
		}
	}
	return nil, fmt.Errorf("Calendar for path: %s not found", path)
}

func (s *backend) CalendarHomeSetPath(ctx context.Context) (string, error) {
	return "/user/calendars/", nil
}

func (s *backend) CurrentUserPrincipal(ctx context.Context) (string, error) {
	return "/user/", nil
}

func (s *backend) DeleteCalendarObject(ctx context.Context, path string) error {
	return nil
}

func (s *backend) GetCalendarObject(ctx context.Context, path string, req *caldav.CalendarCompRequest) (*caldav.CalendarObject, error) {
	for _, objs := range s.objectMap {
		for _, obj := range objs {
			if obj.Path == path {
				return &obj, nil
			}
		}
	}
	return nil, fmt.Errorf("Couldn't find calendar object at: %s", path)
}

func (s *backend) PutCalendarObject(
	ctx context.Context,
	path string,
	calendar *ical.Calendar,
	opts *caldav.PutCalendarObjectOptions,
) (*caldav.CalendarObject, error) {
	object := caldav.CalendarObject{
		Path: path,
		Data: calendar,
	}
	s.objectMap[path] = append(s.objectMap[path], object)
	return &object, nil
}

func (s *backend) ListCalendarObjects(
	ctx context.Context,
	path string,
	req *caldav.CalendarCompRequest,
) ([]caldav.CalendarObject, error) {
	return s.objectMap[path], nil
}

func (s *backend) QueryCalendarObjects(
	ctx context.Context,
	path string,
	query *caldav.CalendarQuery,
) ([]caldav.CalendarObject, error) {
	return nil, nil
}

func RunServer() {
	propFindUserPrincipal := `
		<?xml version="1.0" encoding="UTF-8"?>
		<A:propfind xmlns:A="DAV:">
		  <A:prop>
		    <A:current-user-principal/>
		    <A:principal-URL/>
		    <A:resourcetype/>
		  </A:prop>
		</A:propfind>
	`

	calendarB := caldav.Calendar{
		Path:                  "/user/calendars/b",
		SupportedComponentSet: []string{"VTODO"},
	}
	calendars := []caldav.Calendar{
		{Path: "/user/calendars/a"},
		calendarB,
	}
	eventSummary := "This is a todo"
	event := ical.NewEvent()
	event.Name = ical.CompToDo
	event.Props.SetText(ical.PropUID, "46bbf47a-1861-41a3-ae06-8d8268c6d41e")
	event.Props.SetDateTime(ical.PropDateTimeStamp, time.Now())
	event.Props.SetText(ical.PropSummary, eventSummary)
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropProductID, "-//xyz Corp//NONSGML PDA Calendar Version 1.0//EN")
	cal.Children = []*ical.Component{
		event.Component,
	}
	object := caldav.CalendarObject{
		Path: "/user/calendars/b/test.ics",
		Data: cal,
	}
	req := httptest.NewRequest(
		"PROPFIND",
		"/user/calendars/",
		strings.NewReader(propFindUserPrincipal),
	)
	req.Header.Set("Content-Type", "application/xml")

	handler := caldav.Handler{
		Backend: &backend{
			calendars: calendars,
			objectMap: map[string][]caldav.CalendarObject{
				calendarB.Path: {object},
			},
		},
		Prefix: "",
	}
	err := http.ListenAndServe(":8080", &handler)
	if err != nil {
		fmt.Println(err)
	}
}
