package comment

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"mini-instagram/internal/entity"
)

type fakeCommentRepo struct {
	createErr  error
	lastCreate entity.Comment

	count    int64
	countErr error

	listItems []entity.Comment
	listErr   error
	lastLimit int
	lastOff   int

	ownership    entity.CommentOwnership
	ownershipErr error

	softDeleteErr    error
	lastSoftDelID    int64
	lastSoftDelPostI int64

	updateContentErr     error
	lastUpdateContentID  int64
	lastUpdateContentVal string
}

func (f *fakeCommentRepo) Create(ctx context.Context, comment entity.Comment) error {
	f.lastCreate = comment
	return f.createErr
}

func (f *fakeCommentRepo) Count(ctx context.Context, postID int64) (int64, error) {
	return f.count, f.countErr
}

func (f *fakeCommentRepo) List(ctx context.Context, postID int64, limit, offset int) ([]entity.Comment, error) {
	f.lastLimit, f.lastOff = limit, offset
	return f.listItems, f.listErr
}

func (f *fakeCommentRepo) GetForDelete(ctx context.Context, commentID int64) (entity.CommentOwnership, error) {
	return f.ownership, f.ownershipErr
}

func (f *fakeCommentRepo) SoftDelete(ctx context.Context, commentID, postID int64) error {
	f.lastSoftDelID, f.lastSoftDelPostI = commentID, postID
	return f.softDeleteErr
}

func (f *fakeCommentRepo) UpdateContent(ctx context.Context, commentID int64, content string) error {
	f.lastUpdateContentID = commentID
	f.lastUpdateContentVal = content
	return f.updateContentErr
}

func TestCreate_Success(t *testing.T) {
	repo := &fakeCommentRepo{}
	uc := New(repo)

	if err := uc.Create(context.Background(), 1, 2, "hello"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastCreate.UserID != 1 || repo.lastCreate.PostID != 2 || repo.lastCreate.Content != "hello" {
		t.Fatalf("unexpected create call: %+v", repo.lastCreate)
	}
}

func TestCreate_EmptyContent(t *testing.T) {
	repo := &fakeCommentRepo{}
	uc := New(repo)

	err := uc.Create(context.Background(), 1, 2, "   ")
	var vErr *entity.ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected *entity.ValidationError, got %T (%v)", err, err)
	}
}

func TestCreate_TooLong(t *testing.T) {
	repo := &fakeCommentRepo{}
	uc := New(repo)

	err := uc.Create(context.Background(), 1, 2, strings.Repeat("a", MaxContentLength+1))
	var vErr *entity.ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected *entity.ValidationError, got %T (%v)", err, err)
	}
}

func TestCreate_PostNotFound(t *testing.T) {
	repo := &fakeCommentRepo{createErr: entity.ErrPostNotFound}
	uc := New(repo)

	err := uc.Create(context.Background(), 1, 2, "hello")
	if !errors.Is(err, entity.ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
}

func TestList_Success(t *testing.T) {
	now := time.Now()
	repo := &fakeCommentRepo{
		count: 2,
		listItems: []entity.Comment{
			{ID: 1, PostID: 2, UserID: 3, Username: "bob", Content: "hi", CreatedAt: now},
		},
	}
	uc := New(repo)

	list, err := uc.List(context.Background(), 2, 1, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if list.Count != 2 || len(list.Items) != 1 {
		t.Fatalf("unexpected list: %+v", list)
	}
	if list.Items[0].CommentID != 1 || list.Items[0].Username != "bob" {
		t.Fatalf("unexpected item: %+v", list.Items[0])
	}
}

func TestList_PaginationClamping(t *testing.T) {
	repo := &fakeCommentRepo{}
	uc := New(repo)

	if _, err := uc.List(context.Background(), 2, 0, 0); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastLimit != DefaultPerPage || repo.lastOff != 0 {
		t.Fatalf("expected defaults, got limit=%d offset=%d", repo.lastLimit, repo.lastOff)
	}

	if _, err := uc.List(context.Background(), 2, 3, 1000); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastLimit != MaxPerPage {
		t.Fatalf("expected per_page clamped to %d, got %d", MaxPerPage, repo.lastLimit)
	}
}

func TestDelete_ByAuthor(t *testing.T) {
	repo := &fakeCommentRepo{ownership: entity.CommentOwnership{CommentID: 5, PostID: 2, AuthorID: 1, PostOwnerID: 9}}
	uc := New(repo)

	if err := uc.Delete(context.Background(), 1, 5); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastSoftDelID != 5 || repo.lastSoftDelPostI != 2 {
		t.Fatalf("unexpected soft delete call: id=%d post=%d", repo.lastSoftDelID, repo.lastSoftDelPostI)
	}
}

func TestDelete_ByPostOwner(t *testing.T) {
	repo := &fakeCommentRepo{ownership: entity.CommentOwnership{CommentID: 5, PostID: 2, AuthorID: 1, PostOwnerID: 9}}
	uc := New(repo)

	if err := uc.Delete(context.Background(), 9, 5); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDelete_Forbidden(t *testing.T) {
	repo := &fakeCommentRepo{ownership: entity.CommentOwnership{CommentID: 5, PostID: 2, AuthorID: 1, PostOwnerID: 9}}
	uc := New(repo)

	err := uc.Delete(context.Background(), 42, 5)
	if !errors.Is(err, entity.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if repo.lastSoftDelID != 0 {
		t.Fatal("expected soft delete not to be called")
	}
}

func TestDelete_NotFound(t *testing.T) {
	repo := &fakeCommentRepo{ownershipErr: entity.ErrCommentNotFound}
	uc := New(repo)

	err := uc.Delete(context.Background(), 1, 5)
	if !errors.Is(err, entity.ErrCommentNotFound) {
		t.Fatalf("expected ErrCommentNotFound, got %v", err)
	}
}

func TestEdit_ByAuthor(t *testing.T) {
	repo := &fakeCommentRepo{ownership: entity.CommentOwnership{CommentID: 5, PostID: 2, AuthorID: 1, PostOwnerID: 9}}
	uc := New(repo)

	if err := uc.Edit(context.Background(), 1, 5, "updated"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastUpdateContentID != 5 || repo.lastUpdateContentVal != "updated" {
		t.Fatalf("unexpected update call: id=%d content=%q", repo.lastUpdateContentID, repo.lastUpdateContentVal)
	}
}

func TestEdit_ByPostOwner(t *testing.T) {
	repo := &fakeCommentRepo{ownership: entity.CommentOwnership{CommentID: 5, PostID: 2, AuthorID: 1, PostOwnerID: 9}}
	uc := New(repo)

	if err := uc.Edit(context.Background(), 9, 5, "updated"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestEdit_Forbidden(t *testing.T) {
	repo := &fakeCommentRepo{ownership: entity.CommentOwnership{CommentID: 5, PostID: 2, AuthorID: 1, PostOwnerID: 9}}
	uc := New(repo)

	err := uc.Edit(context.Background(), 42, 5, "updated")
	if !errors.Is(err, entity.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if repo.lastUpdateContentID != 0 {
		t.Fatal("expected update not to be called")
	}
}

func TestEdit_NotFound(t *testing.T) {
	repo := &fakeCommentRepo{ownershipErr: entity.ErrCommentNotFound}
	uc := New(repo)

	err := uc.Edit(context.Background(), 1, 5, "updated")
	if !errors.Is(err, entity.ErrCommentNotFound) {
		t.Fatalf("expected ErrCommentNotFound, got %v", err)
	}
}

func TestEdit_EmptyContent(t *testing.T) {
	repo := &fakeCommentRepo{ownership: entity.CommentOwnership{CommentID: 5, PostID: 2, AuthorID: 1, PostOwnerID: 9}}
	uc := New(repo)

	var vErr *entity.ValidationError
	err := uc.Edit(context.Background(), 1, 5, "   ")
	if !errors.As(err, &vErr) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}
