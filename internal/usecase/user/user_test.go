package user

import (
	"context"
	"errors"
	"image"
	"image/jpeg"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mini-instagram/internal/controller/restapi/v1/request"
	"mini-instagram/internal/entity"
	"mini-instagram/pkg/storage"
)

type nopLogger struct{}

func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Error(string, ...any) {}

type fakeUserRepo struct {
	byID    entity.User
	byIDErr error

	usernameExists bool
	existsErr      error

	updated   entity.User
	updateErr error

	followErr    error
	unfollowErr  error
	followedWith struct {
		followerID, followingID int64
	}

	searchCount   int64
	searchResults []entity.User
	searchErr     error
	searchArgs    struct {
		likePattern, exactMatch string
	}
}

func (f *fakeUserRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	return false, nil
}

func (f *fakeUserRepo) UsernameExists(ctx context.Context, username string) (bool, error) {
	return f.usernameExists, f.existsErr
}

func (f *fakeUserRepo) FindByEmail(ctx context.Context, email string) (entity.User, error) {
	return entity.User{}, entity.ErrNotFound
}

func (f *fakeUserRepo) FindByID(ctx context.Context, id int64) (entity.User, error) {
	if f.byIDErr != nil {
		return entity.User{}, f.byIDErr
	}
	return f.byID, nil
}

func (f *fakeUserRepo) Create(ctx context.Context, user entity.User) (entity.User, error) {
	return entity.User{}, nil
}

func (f *fakeUserRepo) Update(ctx context.Context, user entity.User) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updated = user
	return nil
}

func (f *fakeUserRepo) GetProfileStats(ctx context.Context, userID int64) (int64, int64, int64, error) {
	return 0, 0, 0, nil
}

func (f *fakeUserRepo) IsFollowing(ctx context.Context, followerID, followingID int64) (bool, error) {
	return false, nil
}

func (f *fakeUserRepo) Follow(ctx context.Context, followerID, followingID int64) error {
	f.followedWith.followerID = followerID
	f.followedWith.followingID = followingID
	return f.followErr
}

func (f *fakeUserRepo) Unfollow(ctx context.Context, followerID, followingID int64) error {
	return f.unfollowErr
}

func (f *fakeUserRepo) CountSearch(ctx context.Context, likePattern string) (int64, error) {
	return f.searchCount, f.searchErr
}

func (f *fakeUserRepo) Search(ctx context.Context, likePattern, exactMatch string, limit, offset int) ([]entity.User, error) {
	f.searchArgs.likePattern = likePattern
	f.searchArgs.exactMatch = exactMatch
	return f.searchResults, f.searchErr
}

type fakePostRepo struct{}

func (f *fakePostRepo) Create(ctx context.Context, post entity.Post, hashtags []string) (entity.Post, error) {
	return entity.Post{}, nil
}
func (f *fakePostRepo) CountByUser(ctx context.Context, userID int64) (int64, error) { return 0, nil }
func (f *fakePostRepo) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]entity.Post, error) {
	return nil, nil
}
func (f *fakePostRepo) CountFeed(ctx context.Context, callerID int64) (int64, error) { return 0, nil }
func (f *fakePostRepo) Like(ctx context.Context, userID, postID int64) error         { return nil }
func (f *fakePostRepo) Unlike(ctx context.Context, userID, postID int64) error       { return nil }
func (f *fakePostRepo) GetByID(ctx context.Context, postID int64) (entity.PostDetail, error) {
	return entity.PostDetail{}, nil
}
func (f *fakePostRepo) IsLiked(ctx context.Context, userID, postID int64) (bool, error) {
	return false, nil
}
func (f *fakePostRepo) GetForDelete(ctx context.Context, postID int64) (entity.Post, error) {
	return entity.Post{}, nil
}
func (f *fakePostRepo) SoftDelete(ctx context.Context, postID int64) error { return nil }
func (f *fakePostRepo) UpdateCaption(ctx context.Context, postID int64, caption string, hashtags []string) error {
	return nil
}
func (f *fakePostRepo) ListFeed(ctx context.Context, callerID int64, limit, offset int) ([]entity.FeedPost, error) {
	return nil, nil
}
func (f *fakePostRepo) GetOwner(ctx context.Context, postID int64) (int64, error) { return 0, nil }
func (f *fakePostRepo) NotifyLike(ctx context.Context, ownerID, actorID, postID int64) error {
	return nil
}
func (f *fakePostRepo) CountLikesBatch(ctx context.Context, postIDs []int64) (map[int64]int64, error) {
	return nil, nil
}
func (f *fakePostRepo) InsertLikeRow(ctx context.Context, userID, postID int64) error { return nil }
func (f *fakePostRepo) DeleteLikeRow(ctx context.Context, userID, postID int64) error { return nil }
func (f *fakePostRepo) UpdateLikeCount(ctx context.Context, postID, count int64) error {
	return nil
}

func newTestStorage(t *testing.T) *storage.Storage {
	t.Helper()
	return storage.New(t.TempDir())
}

func mustOpenTestImage(t *testing.T) (*os.File, *multipart.FileHeader) {
	t.Helper()

	f, err := os.CreateTemp("", "test-avatar-*.jpg")
	if err != nil {
		t.Fatalf("create temp image: %v", err)
	}
	t.Cleanup(func() { os.Remove(f.Name()) })

	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
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

func TestUpdateProfile_ValidationError(t *testing.T) {
	users := &fakeUserRepo{byID: entity.User{ID: 1, Username: "janedoe"}}
	uc := New(users, &fakePostRepo{}, newTestStorage(t), nopLogger{})

	err := uc.UpdateProfile(context.Background(), request.UpdateProfile{
		UserID:   1,
		Username: "ab", // too short
		FullName: "Jane Doe",
	})

	var vErr *entity.ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected *entity.ValidationError, got %T (%v)", err, err)
	}
}

func TestUpdateProfile_UsernameTaken(t *testing.T) {
	users := &fakeUserRepo{byID: entity.User{ID: 1, Username: "janedoe"}, usernameExists: true}
	uc := New(users, &fakePostRepo{}, newTestStorage(t), nopLogger{})

	err := uc.UpdateProfile(context.Background(), request.UpdateProfile{
		UserID:   1,
		Username: "newname",
		FullName: "Jane Doe",
	})
	if !errors.Is(err, entity.ErrUsernameTaken) {
		t.Fatalf("expected ErrUsernameTaken, got %v", err)
	}
}

func TestUpdateProfile_UserNotFound(t *testing.T) {
	users := &fakeUserRepo{byIDErr: entity.ErrNotFound}
	uc := New(users, &fakePostRepo{}, newTestStorage(t), nopLogger{})

	err := uc.UpdateProfile(context.Background(), request.UpdateProfile{
		UserID:   1,
		Username: "janedoe",
		FullName: "Jane Doe",
	})
	if !errors.Is(err, entity.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateProfile_SameUsernameSkipsUniquenessCheck(t *testing.T) {
	users := &fakeUserRepo{byID: entity.User{ID: 1, Username: "janedoe"}, existsErr: errors.New("should not be called")}
	uc := New(users, &fakePostRepo{}, newTestStorage(t), nopLogger{})

	err := uc.UpdateProfile(context.Background(), request.UpdateProfile{
		UserID:   1,
		Username: "janedoe",
		FullName: "Jane Doe",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUpdateProfile_SanitizesAndUpdates(t *testing.T) {
	users := &fakeUserRepo{byID: entity.User{ID: 1, Username: "janedoe", FullName: "Old Name"}}
	uc := New(users, &fakePostRepo{}, newTestStorage(t), nopLogger{})

	err := uc.UpdateProfile(context.Background(), request.UpdateProfile{
		UserID:   1,
		Username: "janedoe",
		FullName: "<script>alert(1)</script>Jane Doe",
		Bio:      "  hello  ",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if strings.Contains(users.updated.FullName, "<script>") {
		t.Fatalf("expected full_name to be sanitized, got %q", users.updated.FullName)
	}
	if users.updated.Bio != "hello" {
		t.Fatalf("expected trimmed bio, got %q", users.updated.Bio)
	}
}

func TestUpdateProfile_WithAvatarReplacesOldOne(t *testing.T) {
	st := newTestStorage(t)
	users := &fakeUserRepo{byID: entity.User{ID: 1, Username: "janedoe", FullName: "Jane Doe", AvatarPath: "avatars/old.jpg"}}
	uc := New(users, &fakePostRepo{}, st, nopLogger{})

	// seed the "old" avatar so deletion can be observed
	if _, err := st.Save("avatars/old.jpg", strings.NewReader("old")); err != nil {
		t.Fatalf("seed old avatar: %v", err)
	}

	file, header := mustOpenTestImage(t)
	defer file.Close()

	err := uc.UpdateProfile(context.Background(), request.UpdateProfile{
		UserID:       1,
		Username:     "janedoe",
		FullName:     "Jane Doe",
		Avatar:       file,
		AvatarHeader: header,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if users.updated.AvatarPath == "" || users.updated.AvatarPath == "avatars/old.jpg" {
		t.Fatalf("expected a new avatar path, got %q", users.updated.AvatarPath)
	}
	if _, err := os.Stat(st.FullPath("avatars/old.jpg")); !os.IsNotExist(err) {
		t.Fatalf("expected old avatar to be deleted, stat err: %v", err)
	}
	if _, err := os.Stat(st.FullPath(users.updated.AvatarPath)); err != nil {
		t.Fatalf("expected new avatar to exist: %v", err)
	}
}

func TestUpdateProfile_InvalidAvatar(t *testing.T) {
	users := &fakeUserRepo{byID: entity.User{ID: 1, Username: "janedoe", FullName: "Jane Doe"}}
	uc := New(users, &fakePostRepo{}, newTestStorage(t), nopLogger{})

	file, err := os.CreateTemp("", "not-an-image-*.txt")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(file.Name())
	defer file.Close()

	header := &multipart.FileHeader{
		Filename: "not-an-image.txt",
		Header:   map[string][]string{"Content-Type": {"text/plain"}},
		Size:     4,
	}

	updateErr := uc.UpdateProfile(context.Background(), request.UpdateProfile{
		UserID:       1,
		Username:     "janedoe",
		FullName:     "Jane Doe",
		Avatar:       file,
		AvatarHeader: header,
	})
	if updateErr == nil {
		t.Fatal("expected error for non-image avatar")
	}
}

func TestUpdateProfile_CleansUpNewAvatarOnUpdateFailure(t *testing.T) {
	st := newTestStorage(t)
	users := &fakeUserRepo{
		byID:      entity.User{ID: 1, Username: "janedoe", FullName: "Jane Doe"},
		updateErr: errors.New("db failure"),
	}
	uc := New(users, &fakePostRepo{}, st, nopLogger{})

	file, header := mustOpenTestImage(t)
	defer file.Close()

	err := uc.UpdateProfile(context.Background(), request.UpdateProfile{
		UserID:       1,
		Username:     "janedoe",
		FullName:     "Jane Doe",
		Avatar:       file,
		AvatarHeader: header,
	})
	if err == nil {
		t.Fatal("expected error")
	}

	entries, readErr := os.ReadDir(st.FullPath("avatars"))
	if readErr != nil && !os.IsNotExist(readErr) {
		t.Fatalf("read avatars dir: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("expected newly saved avatar to be cleaned up, found %d entries", len(entries))
	}
}

func TestSearchUsers_EmptyQuery(t *testing.T) {
	users := &fakeUserRepo{}
	uc := New(users, &fakePostRepo{}, newTestStorage(t), nopLogger{})

	_, err := uc.SearchUsers(context.Background(), "   ", 1, 10)
	var vErr *entity.ValidationError
	if !errors.As(err, &vErr) || vErr.Field != "q" {
		t.Fatalf("expected validation error on field q, got %v", err)
	}
}

func TestSearchUsers_TooLong(t *testing.T) {
	users := &fakeUserRepo{}
	uc := New(users, &fakePostRepo{}, newTestStorage(t), nopLogger{})

	_, err := uc.SearchUsers(context.Background(), strings.Repeat("a", 33), 1, 10)
	var vErr *entity.ValidationError
	if !errors.As(err, &vErr) || vErr.Field != "q" {
		t.Fatalf("expected validation error on field q, got %v", err)
	}
}

func TestSearchUsers_EscapesWildcardsAndLowercasesExactMatch(t *testing.T) {
	users := &fakeUserRepo{searchCount: 1, searchResults: []entity.User{{ID: 1, Username: "jane_doe", FullName: "Jane Doe"}}}
	uc := New(users, &fakePostRepo{}, newTestStorage(t), nopLogger{})

	result, err := uc.SearchUsers(context.Background(), "Jane%_", 1, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Count != 1 || len(result.Items) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if users.searchArgs.likePattern != `Jane\%\_` {
		t.Fatalf("expected escaped like pattern, got %q", users.searchArgs.likePattern)
	}
	if users.searchArgs.exactMatch != "jane%_" {
		t.Fatalf("expected lowercased exact match, got %q", users.searchArgs.exactMatch)
	}
}

func TestFollow_SelfFollow(t *testing.T) {
	users := &fakeUserRepo{byID: entity.User{ID: 1, IsActive: true}}
	uc := New(users, &fakePostRepo{}, newTestStorage(t), nopLogger{})

	err := uc.Follow(context.Background(), 1, 1)
	if !errors.Is(err, entity.ErrSelfFollow) {
		t.Fatalf("expected ErrSelfFollow, got %v", err)
	}
}

func TestFollow_TargetNotFound(t *testing.T) {
	users := &fakeUserRepo{byIDErr: entity.ErrNotFound}
	uc := New(users, &fakePostRepo{}, newTestStorage(t), nopLogger{})

	err := uc.Follow(context.Background(), 1, 2)
	if !errors.Is(err, entity.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestFollow_TargetInactive(t *testing.T) {
	users := &fakeUserRepo{byID: entity.User{ID: 2, IsActive: false}}
	uc := New(users, &fakePostRepo{}, newTestStorage(t), nopLogger{})

	err := uc.Follow(context.Background(), 1, 2)
	if !errors.Is(err, entity.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestFollow_Success(t *testing.T) {
	users := &fakeUserRepo{byID: entity.User{ID: 2, IsActive: true}}
	uc := New(users, &fakePostRepo{}, newTestStorage(t), nopLogger{})

	if err := uc.Follow(context.Background(), 1, 2); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if users.followedWith.followerID != 1 || users.followedWith.followingID != 2 {
		t.Fatalf("expected repo Follow called with (1, 2), got %+v", users.followedWith)
	}
}

func TestFollow_AlreadyFollowing(t *testing.T) {
	users := &fakeUserRepo{byID: entity.User{ID: 2, IsActive: true}, followErr: entity.ErrAlreadyFollowing}
	uc := New(users, &fakePostRepo{}, newTestStorage(t), nopLogger{})

	err := uc.Follow(context.Background(), 1, 2)
	if !errors.Is(err, entity.ErrAlreadyFollowing) {
		t.Fatalf("expected ErrAlreadyFollowing, got %v", err)
	}
}

func TestUnfollow_NotFollowing(t *testing.T) {
	users := &fakeUserRepo{byID: entity.User{ID: 2, IsActive: true}, unfollowErr: entity.ErrNotFollowing}
	uc := New(users, &fakePostRepo{}, newTestStorage(t), nopLogger{})

	err := uc.Unfollow(context.Background(), 1, 2)
	if !errors.Is(err, entity.ErrNotFollowing) {
		t.Fatalf("expected ErrNotFollowing, got %v", err)
	}
}

func TestUnfollow_Success(t *testing.T) {
	users := &fakeUserRepo{byID: entity.User{ID: 2, IsActive: true}}
	uc := New(users, &fakePostRepo{}, newTestStorage(t), nopLogger{})

	if err := uc.Unfollow(context.Background(), 1, 2); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
