package model

import (
	"fmt"
	"time"
)

type Transcription struct {
	Id        int64  `db:"id"`
	TimeStamp int64  `db:"ts"`
	Author    string `db:"user_name"`
	AuthorID  int64  `db:"-"`
	Data      string `db:"data"`
}

type TranscriptionListItem struct {
	Id        int64  `db:"id"`
	TimeStamp int64  `db:"ts"`
	Author    string `db:"user_name"`
}

func (tli TranscriptionListItem) String() string {
	return fmt.Sprintf("Транскрипция встречи: %d Время: %s Автор: %s", tli.Id, time.Unix(tli.TimeStamp, 0).String(), tli.Author)
}

func (tr *Transcription) IsEmpty() bool {
	return *tr == Transcription{}
}
