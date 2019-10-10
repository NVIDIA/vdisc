// Copyright © 2018 NVIDIA Corporation

package datetime

import (
	"fmt"
	"log"
	"time"
)

type DecDateTime [17]byte

func NewDecDateTime(t time.Time) DecDateTime {
	t = t.UTC()

	var result DecDateTime
	copy(result[0:4], fmt.Sprintf("%04d", t.Year()))
	copy(result[4:6], fmt.Sprintf("%02d", t.Month()))
	copy(result[6:8], fmt.Sprintf("%02d", t.Day()))
	copy(result[8:10], fmt.Sprintf("%02d", t.Hour()))
	copy(result[10:12], fmt.Sprintf("%02d", t.Minute()))
	copy(result[12:14], fmt.Sprintf("%02d", t.Second()))
	copy(result[14:16], fmt.Sprintf("%02d", t.Nanosecond()/10000000)) // Hundredths of a second
	result[16] = 50

	return result
}

func (ddt DecDateTime) Timestamp() time.Time {
	t, err := time.Parse("20060102150405", string(ddt[0:14]))
	if err != nil {
		log.Printf("Failed to parse DecDateTime, %s, defaulting to unix epoch.", err)
		return time.Unix(0, 0).UTC()
	}

	// TODO: parse nanoseconds and TZ
	return t.UTC()
}

type EntryDateTime [7]byte

func NewEntryDateTime(t time.Time) EntryDateTime {
	t = t.UTC()
	var result EntryDateTime
	result[0] = byte(t.Year() - 1900)
	result[1] = byte(t.Month())
	result[2] = byte(t.Day())
	result[3] = byte(t.Hour())
	result[4] = byte(t.Minute())
	result[5] = byte(t.Second())
	result[6] = 50
	return result
}

func (edt EntryDateTime) Timestamp() time.Time {
	year := int(edt[0]) + 1900
	month := time.Month(edt[1])
	day := int(edt[2])
	hour := int(edt[3])
	minute := int(edt[4])
	second := int(edt[5])

	// TODO: TZ
	return time.Date(year, month, day, hour, minute, second, 0, time.UTC)

}

var (
	MaxDecDateTime = DecDateTime([17]byte{48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 0})
)
