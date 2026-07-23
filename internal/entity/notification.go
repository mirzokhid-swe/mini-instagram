package entity

import "time"

type Notification struct {
	ID            int64
	ActionType    string
	ActorID       int64
	ActorUsername string
	PostID        int64
	IsRead        bool
	CreatedAt     time.Time
}
