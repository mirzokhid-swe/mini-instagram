package post

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mini-instagram/internal/controller/restapi/v1/request"
	"mini-instagram/internal/entity"
	"mini-instagram/pkg/storage"
)

type fakePostRepo struct {
	created         entity.Post
	createdHashtags []string
	err             error

	feedCount int64
	feedPosts []entity.FeedPost
	feedErr   error

	lastFeedLimit  int
	lastFeedOffset int

	likeErr, unlikeErr                 error
	lastLikeUserID, lastLikePostID     int64
	lastUnlikeUserID, lastUnlikePostID int64

	getByIDPost   entity.PostDetail
	getByIDErr    error
	isLiked       bool
	isLikedErr    error
	getForDeleteP entity.Post
	getForDelErr  error
	softDeleteErr error
	lastSoftDelID int64
}

func (f *fakePostRepo) Create(ctx context.Context, post entity.Post, hashtags []string) (entity.Post, error) {
	if f.err != nil {
		return entity.Post{}, f.err
	}
	f.created = post
	f.created.ID = 1
	f.createdHashtags = hashtags
	return f.created, nil
}

func (f *fakePostRepo) CountByUser(ctx context.Context, userID int64) (int64, error) {
	return 0, nil
}

func (f *fakePostRepo) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]entity.Post, error) {
	return nil, nil
}

func (f *fakePostRepo) CountFeed(ctx context.Context, callerID int64) (int64, error) {
	if f.feedErr != nil {
		return 0, f.feedErr
	}
	return f.feedCount, nil
}

func (f *fakePostRepo) ListFeed(ctx context.Context, callerID int64, limit, offset int) ([]entity.FeedPost, error) {
	if f.feedErr != nil {
		return nil, f.feedErr
	}
	f.lastFeedLimit = limit
	f.lastFeedOffset = offset
	return f.feedPosts, nil
}

func (f *fakePostRepo) Like(ctx context.Context, userID, postID int64) error {
	f.lastLikeUserID, f.lastLikePostID = userID, postID
	return f.likeErr
}

func (f *fakePostRepo) Unlike(ctx context.Context, userID, postID int64) error {
	f.lastUnlikeUserID, f.lastUnlikePostID = userID, postID
	return f.unlikeErr
}

func (f *fakePostRepo) GetByID(ctx context.Context, postID int64) (entity.PostDetail, error) {
	return f.getByIDPost, f.getByIDErr
}

func (f *fakePostRepo) IsLiked(ctx context.Context, userID, postID int64) (bool, error) {
	return f.isLiked, f.isLikedErr
}

func (f *fakePostRepo) GetForDelete(ctx context.Context, postID int64) (entity.Post, error) {
	return f.getForDeleteP, f.getForDelErr
}

func (f *fakePostRepo) SoftDelete(ctx context.Context, postID int64) error {
	f.lastSoftDelID = postID
	return f.softDeleteErr
}

type fakeHashtagRepo struct {
	count int64
	posts []entity.HashtagPost
	err   error
}

func (f *fakeHashtagRepo) CountByTag(ctx context.Context, tag string) (int64, error) {
	return f.count, f.err
}

func (f *fakeHashtagRepo) ListByTag(ctx context.Context, tag string, limit, offset int) ([]entity.HashtagPost, error) {
	return f.posts, f.err
}

func newTestStorage(t *testing.T) *storage.Storage {
	t.Helper()
	dir := t.TempDir()
	return storage.New(dir)
}

func mustOpenTestImage(t *testing.T) (*os.File, *multipart.FileHeader) {
	t.Helper()

	f, err := os.CreateTemp("", "test-image-*.jpg")
	if err != nil {
		t.Fatalf("create temp image: %v", err)
	}
	defer os.Remove(f.Name())

	img := image.NewRGBA(image.Rect(0, 0, 500, 300))
	if err := jpeg.Encode(f, img, nil); err != nil {
		t.Fatalf("encode test image: %v", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatalf("seek temp image: %v", err)
	}

	fi, err := f.Stat()
	if err != nil {
		t.Fatalf("stat temp image: %v", err)
	}

	header := &multipart.FileHeader{
		Filename: filepath.Base(f.Name()),
		Header:   map[string][]string{"Content-Type": {"image/jpeg"}},
		Size:     fi.Size(),
	}
	return f, header
}

type nopLogger struct{}

func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Error(string, ...any) {}

func TestCreatePost_Success(t *testing.T) {
	repo := &fakePostRepo{}
	st := newTestStorage(t)
	uc := New(repo, &fakeHashtagRepo{}, st, nopLogger{})

	file, header := mustOpenTestImage(t)
	defer file.Close()

	err := uc.Create(context.Background(), request.CreatePost{UserID: 1, Caption: "hello", File: file, Header: header})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.created.ID != 1 {
		t.Fatalf("expected post id 1, got %d", repo.created.ID)
	}
	if repo.created.UserID != 1 {
		t.Fatalf("expected user id 1, got %d", repo.created.UserID)
	}
	if repo.created.Caption != "hello" {
		t.Fatalf("expected caption hello, got %s", repo.created.Caption)
	}
	if repo.created.ImagePath == "" {
		t.Fatal("expected non-empty image path")
	}
	if repo.created.ThumbnailPath == "" {
		t.Fatal("expected non-empty thumbnail path")
	}

	if _, err := os.Stat(st.FullPath(repo.created.ImagePath)); err != nil {
		t.Fatalf("image file missing: %v", err)
	}
	if _, err := os.Stat(st.FullPath(repo.created.ThumbnailPath)); err != nil {
		t.Fatalf("thumbnail file missing: %v", err)
	}
}

type fakeFile struct {
	io.ReadSeeker
}

func (fakeFile) ReadAt(p []byte, off int64) (int, error) { return 0, io.EOF }
func (fakeFile) Close() error                            { return nil }

func TestCreatePost_InvalidImage(t *testing.T) {
	repo := &fakePostRepo{}
	st := newTestStorage(t)
	uc := New(repo, &fakeHashtagRepo{}, st, nopLogger{})

	file := fakeFile{ReadSeeker: strings.NewReader("not an image")}
	header := &multipart.FileHeader{
		Filename: "test.txt",
		Header:   map[string][]string{"Content-Type": {"text/plain"}},
		Size:     12,
	}

	err := uc.Create(context.Background(), request.CreatePost{UserID: 1, Caption: "", File: file, Header: header})
	if err == nil {
		t.Fatal("expected error for invalid image")
	}
}

func TestCreatePost_CaptionSanitizesHTML(t *testing.T) {
	repo := &fakePostRepo{}
	st := newTestStorage(t)
	uc := New(repo, &fakeHashtagRepo{}, st, nopLogger{})

	file, header := mustOpenTestImage(t)
	defer file.Close()

	err := uc.Create(context.Background(), request.CreatePost{UserID: 1, Caption: "<script>alert(1)</script>hello", File: file, Header: header})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if strings.Contains(repo.created.Caption, "<script>") {
		t.Fatalf("expected HTML tags to be stripped, got %q", repo.created.Caption)
	}
	if repo.created.Caption != "hello" {
		t.Fatalf("expected caption 'hello', got %q", repo.created.Caption)
	}
}

func TestCreatePost_CaptionTooLong(t *testing.T) {
	repo := &fakePostRepo{}
	st := newTestStorage(t)
	uc := New(repo, &fakeHashtagRepo{}, st, nopLogger{})

	file, header := mustOpenTestImage(t)
	defer file.Close()

	err := uc.Create(context.Background(), request.CreatePost{UserID: 1, Caption: strings.Repeat("a", MaxCaptionLength+1), File: file, Header: header})
	if err == nil {
		t.Fatal("expected error for long caption")
	}
}

func TestCreatePost_CleanupOnRepoFailure(t *testing.T) {
	repo := &fakePostRepo{err: errors.New("db failure")}
	st := newTestStorage(t)
	uc := New(repo, &fakeHashtagRepo{}, st, nopLogger{})

	file, header := mustOpenTestImage(t)
	defer file.Close()

	err := uc.Create(context.Background(), request.CreatePost{UserID: 1, Caption: "hello", File: file, Header: header})
	if err == nil {
		t.Fatal("expected error")
	}

	entries, err := os.ReadDir(st.FullPath("posts"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read posts dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected cleanup of image file, found %d entries", len(entries))
	}
}

func TestGetFeed_Success(t *testing.T) {
	now := time.Now()
	repo := &fakePostRepo{
		feedCount: 2,
		feedPosts: []entity.FeedPost{
			{ID: 2, UserID: 5, Username: "bob", Caption: "hi", ImagePath: "posts/b.jpg", LikeCount: 3, CommentCount: 1, CreatedAt: now},
			{ID: 1, UserID: 5, Username: "bob", Caption: "hello", ImagePath: "posts/a.jpg", CreatedAt: now.Add(-time.Hour)},
		},
	}
	uc := New(repo, &fakeHashtagRepo{}, newTestStorage(t), nopLogger{})

	feed, err := uc.GetFeed(context.Background(), 1, 1, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if feed.Count != 2 {
		t.Fatalf("expected count 2, got %d", feed.Count)
	}
	if len(feed.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(feed.Items))
	}
	if feed.Items[0].PostID != 2 || feed.Items[0].Username != "bob" {
		t.Fatalf("unexpected first item: %+v", feed.Items[0])
	}
	if feed.Items[0].LikesCount != 3 || feed.Items[0].CommentsCount != 1 {
		t.Fatalf("expected counters to be carried over, got %+v", feed.Items[0])
	}
}

func TestGetFeed_EmptyWhenFollowingNobody(t *testing.T) {
	repo := &fakePostRepo{}
	uc := New(repo, &fakeHashtagRepo{}, newTestStorage(t), nopLogger{})

	feed, err := uc.GetFeed(context.Background(), 1, 1, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if feed.Count != 0 || len(feed.Items) != 0 {
		t.Fatalf("expected empty feed, got %+v", feed)
	}
}

func TestGetFeed_PaginationClamping(t *testing.T) {
	repo := &fakePostRepo{}
	uc := New(repo, &fakeHashtagRepo{}, newTestStorage(t), nopLogger{})

	if _, err := uc.GetFeed(context.Background(), 1, 0, 0); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastFeedLimit != DefaultPerPage || repo.lastFeedOffset != 0 {
		t.Fatalf("expected defaults to apply, got limit=%d offset=%d", repo.lastFeedLimit, repo.lastFeedOffset)
	}

	if _, err := uc.GetFeed(context.Background(), 1, 3, 1000); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastFeedLimit != MaxPerPage {
		t.Fatalf("expected per_page to be clamped to %d, got %d", MaxPerPage, repo.lastFeedLimit)
	}
	wantOffset := (3 - 1) * MaxPerPage
	if repo.lastFeedOffset != wantOffset {
		t.Fatalf("expected offset %d, got %d", wantOffset, repo.lastFeedOffset)
	}
}

func TestGetFeed_RepoError(t *testing.T) {
	repo := &fakePostRepo{feedErr: errors.New("db failure")}
	uc := New(repo, &fakeHashtagRepo{}, newTestStorage(t), nopLogger{})

	if _, err := uc.GetFeed(context.Background(), 1, 1, 10); err == nil {
		t.Fatal("expected error")
	}
}

func TestLike_Success(t *testing.T) {
	repo := &fakePostRepo{}
	uc := New(repo, &fakeHashtagRepo{}, newTestStorage(t), nopLogger{})

	if err := uc.Like(context.Background(), 1, 2); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastLikeUserID != 1 || repo.lastLikePostID != 2 {
		t.Fatalf("expected repo called with (1, 2), got (%d, %d)", repo.lastLikeUserID, repo.lastLikePostID)
	}
}

func TestLike_PostNotFound(t *testing.T) {
	repo := &fakePostRepo{likeErr: entity.ErrPostNotFound}
	uc := New(repo, &fakeHashtagRepo{}, newTestStorage(t), nopLogger{})

	err := uc.Like(context.Background(), 1, 2)
	if !errors.Is(err, entity.ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
}

func TestUnlike_Success(t *testing.T) {
	repo := &fakePostRepo{}
	uc := New(repo, &fakeHashtagRepo{}, newTestStorage(t), nopLogger{})

	if err := uc.Unlike(context.Background(), 1, 2); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastUnlikeUserID != 1 || repo.lastUnlikePostID != 2 {
		t.Fatalf("expected repo called with (1, 2), got (%d, %d)", repo.lastUnlikeUserID, repo.lastUnlikePostID)
	}
}

func TestUnlike_NotLiked(t *testing.T) {
	repo := &fakePostRepo{unlikeErr: entity.ErrNotLiked}
	uc := New(repo, &fakeHashtagRepo{}, newTestStorage(t), nopLogger{})

	err := uc.Unlike(context.Background(), 1, 2)
	if !errors.Is(err, entity.ErrNotLiked) {
		t.Fatalf("expected ErrNotLiked, got %v", err)
	}
}

func TestGetByID_Success(t *testing.T) {
	now := time.Now()
	repo := &fakePostRepo{
		getByIDPost: entity.PostDetail{ID: 2, UserID: 5, Username: "bob", Caption: "hi", ImagePath: "posts/b.jpg", LikeCount: 3, CommentCount: 1, CreatedAt: now},
		isLiked:     true,
	}
	uc := New(repo, &fakeHashtagRepo{}, newTestStorage(t), nopLogger{})

	post, err := uc.GetByID(context.Background(), 1, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if post.PostID != 2 || post.Username != "bob" || !post.IsLiked {
		t.Fatalf("unexpected post detail: %+v", post)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	repo := &fakePostRepo{getByIDErr: entity.ErrPostNotFound}
	uc := New(repo, &fakeHashtagRepo{}, newTestStorage(t), nopLogger{})

	_, err := uc.GetByID(context.Background(), 1, 2)
	if !errors.Is(err, entity.ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
}

func TestDelete_Success(t *testing.T) {
	st := newTestStorage(t)
	if _, err := st.Save("posts/img.jpg", strings.NewReader("img")); err != nil {
		t.Fatalf("seed image: %v", err)
	}
	if _, err := st.Save("thumbnails/thumb.jpg", strings.NewReader("thumb")); err != nil {
		t.Fatalf("seed thumbnail: %v", err)
	}

	repo := &fakePostRepo{getForDeleteP: entity.Post{ID: 2, UserID: 1, ImagePath: "posts/img.jpg", ThumbnailPath: "thumbnails/thumb.jpg"}}
	uc := New(repo, &fakeHashtagRepo{}, st, nopLogger{})

	if err := uc.Delete(context.Background(), 1, 2); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastSoftDelID != 2 {
		t.Fatalf("expected post 2 soft-deleted, got %d", repo.lastSoftDelID)
	}
	if _, err := os.Stat(st.FullPath("posts/img.jpg")); !os.IsNotExist(err) {
		t.Fatalf("expected image file to be removed, stat err: %v", err)
	}
	if _, err := os.Stat(st.FullPath("thumbnails/thumb.jpg")); !os.IsNotExist(err) {
		t.Fatalf("expected thumbnail file to be removed, stat err: %v", err)
	}
}

func TestDelete_NotOwner(t *testing.T) {
	repo := &fakePostRepo{getForDeleteP: entity.Post{ID: 2, UserID: 99, ImagePath: "posts/img.jpg"}}
	uc := New(repo, &fakeHashtagRepo{}, newTestStorage(t), nopLogger{})

	err := uc.Delete(context.Background(), 1, 2)
	if !errors.Is(err, entity.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if repo.lastSoftDelID != 0 {
		t.Fatal("expected soft delete not to be called")
	}
}

func TestDelete_NotFound(t *testing.T) {
	repo := &fakePostRepo{getForDelErr: entity.ErrPostNotFound}
	uc := New(repo, &fakeHashtagRepo{}, newTestStorage(t), nopLogger{})

	err := uc.Delete(context.Background(), 1, 2)
	if !errors.Is(err, entity.ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
}

func TestCreatePost_ParsesHashtags(t *testing.T) {
	repo := &fakePostRepo{}
	st := newTestStorage(t)
	uc := New(repo, &fakeHashtagRepo{}, st, nopLogger{})

	file, header := mustOpenTestImage(t)
	defer file.Close()

	err := uc.Create(context.Background(), request.CreatePost{
		UserID:  1,
		Caption: "sunset #Beach #beach #beach_life",
		File:    file,
		Header:  header,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := []string{"beach", "beach_life"}
	if len(repo.createdHashtags) != len(want) {
		t.Fatalf("expected hashtags %v, got %v", want, repo.createdHashtags)
	}
	for i, tag := range want {
		if repo.createdHashtags[i] != tag {
			t.Fatalf("expected hashtags %v, got %v", want, repo.createdHashtags)
		}
	}
}

func TestCreatePost_HashtagCapAndLengthLimit(t *testing.T) {
	var caption strings.Builder
	for i := range 35 {
		fmt.Fprintf(&caption, "#tag%d ", i)
	}
	caption.WriteString("#" + strings.Repeat("a", 65))

	tags := parseHashtags(caption.String())
	if len(tags) != MaxHashtags {
		t.Fatalf("expected %d hashtags, got %d", MaxHashtags, len(tags))
	}
	for _, tag := range tags {
		if len(tag) > MaxHashtagLength {
			t.Fatalf("expected hashtag %q to be at most %d chars", tag, MaxHashtagLength)
		}
	}
}

func TestSearchByTag_StripsHashAndLowercases(t *testing.T) {
	repo := &fakeHashtagRepo{count: 1, posts: []entity.HashtagPost{{ID: 1, UserID: 2, Username: "bob", Caption: "hi", ThumbnailPath: "t.jpg"}}}
	uc := New(&fakePostRepo{}, repo, newTestStorage(t), nopLogger{})

	result, err := uc.SearchByTag(context.Background(), "#Beach", 1, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Count != 1 || len(result.Items) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestSearchByTag_EmptyTag(t *testing.T) {
	uc := New(&fakePostRepo{}, &fakeHashtagRepo{}, newTestStorage(t), nopLogger{})

	_, err := uc.SearchByTag(context.Background(), "   #  ", 1, 10)
	var vErr *entity.ValidationError
	if !errors.As(err, &vErr) || vErr.Field != "tag" {
		t.Fatalf("expected validation error on field tag, got %v", err)
	}
}

func TestSearchByTag_UnknownTagReturnsEmpty(t *testing.T) {
	uc := New(&fakePostRepo{}, &fakeHashtagRepo{}, newTestStorage(t), nopLogger{})

	result, err := uc.SearchByTag(context.Background(), "nonexistent", 1, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Count != 0 || len(result.Items) != 0 {
		t.Fatalf("expected empty result, got %+v", result)
	}
}
