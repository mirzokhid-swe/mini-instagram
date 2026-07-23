package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	"mini-instagram/internal/entity"
)

type fakeNotificationRepo struct {
	count    int64
	countErr error

	items   []entity.Notification
	listErr error

	markReadErr error
	markReadArg struct {
		notificationID, userID int64
	}
}

func (f *fakeNotificationRepo) Count(ctx context.Context, userID int64) (int64, error) {
	return f.count, f.countErr
}

func (f *fakeNotificationRepo) List(ctx context.Context, userID int64, limit, offset int) ([]entity.Notification, error) {
	return f.items, f.listErr
}

func (f *fakeNotificationRepo) MarkRead(ctx context.Context, notificationID, userID int64) error {
	f.markReadArg.notificationID = notificationID
	f.markReadArg.userID = userID
	return f.markReadErr
}

func TestList_MapsItems(t *testing.T) {
	now := time.Now()
	repo := &fakeNotificationRepo{
		count: 1,
		items: []entity.Notification{
			{ID: 1, ActionType: "like", ActorID: 2, ActorUsername: "jane", PostID: 3, IsRead: false, CreatedAt: now},
		},
	}
	uc := New(repo)

	list, err := uc.List(context.Background(), 1, 1, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if list.Count != 1 || len(list.Items) != 1 {
		t.Fatalf("unexpected list: %+v", list)
	}
	item := list.Items[0]
	if item.NotificationID != 1 || item.ActionType != "like" || item.ActorUsername != "jane" || item.PostID != 3 {
		t.Fatalf("unexpected item: %+v", item)
	}
}

func TestList_DefaultsAndCaps(t *testing.T) {
	repo := &fakeNotificationRepo{}
	uc := New(repo)

	if _, err := uc.List(context.Background(), 1, 0, 0); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, err := uc.List(context.Background(), 1, 1, 1000); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestMarkRead_NotFound(t *testing.T) {
	repo := &fakeNotificationRepo{markReadErr: entity.ErrNotificationNotFound}
	uc := New(repo)

	err := uc.MarkRead(context.Background(), 1, 2)
	if !errors.Is(err, entity.ErrNotificationNotFound) {
		t.Fatalf("expected ErrNotificationNotFound, got %v", err)
	}
}

func TestMarkRead_Success(t *testing.T) {
	repo := &fakeNotificationRepo{}
	uc := New(repo)

	if err := uc.MarkRead(context.Background(), 1, 2); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.markReadArg.notificationID != 1 || repo.markReadArg.userID != 2 {
		t.Fatalf("unexpected repo call args: %+v", repo.markReadArg)
	}
}
