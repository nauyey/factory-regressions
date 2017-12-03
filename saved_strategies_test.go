package main_test

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	. "github.com/nauyey/factory"
	"github.com/nauyey/factory/def"
)

type testUser struct {
	ID           int64     `factory:"id,primary"`
	Name         string    `factory:"name"`
	NickName     string    `factory:"nick_name"`
	Age          int32     `factory:"age"`
	Country      string    `factory:"country"`
	BirthTime    time.Time `factory:"birth_time"`
	Now          time.Time
	Blogs        []*testBlog
	NotSaveField string
}

type testBlog struct {
	ID       int64  `factory:"id,primary"`
	Title    string `factory:"title"`
	Content  string `factory:"content"`
	AuthorID int64  `factory:"author_id"`
	Author   *testUser
}

type testComment struct {
	ID     int64  `factory:"id,primary"`
	Text   string `factory:"text"`
	BlogID int64  `factory:"blog_id"`
	UserID int64  `factory:"user_id"`
	Blog   *testBlog
	User   *testUser
}

type testStar struct {
	ID     int64 `factory:"id,primary"`
	Count  int   `factory:"count"`
	BlogID int64 `factory:"blog_id"`
	Blog   *testBlog
}

type relation struct {
	Author *testUser
}

type testCommentary struct {
	ID       int64  `factory:"id,primary"`
	Title    string `factory:"title"`
	Content  string `factory:"content"`
	AuthorID int64  `factory:"author_id"`
	R        *relation
	Comment  *testComment
}

func init() {
	db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/test_factory?parseTime=true")
	if err != nil {
		panic(err)
	}
	SetDB(db)
	DebugMode = true
}

func TestCreate(t *testing.T) {
	// define factory
	var birthTime, _ = time.Parse("2006-01-02T15:04:05.000Z", "2000-11-19T00:00:00.000Z")
	var now, _ = time.Parse("2006-01-02T15:04:05.000Z", "2017-11-19T00:00:00.000Z")
	userFactory := def.NewFactory(testUser{}, "test_user",
		def.Field("Name", "test name"),
		def.SequenceField("ID", 1, func(n int64) (interface{}, error) {
			return n, nil
		}),
		def.Field("Now", now),
		def.AfterBuild(func(model interface{}) error {
			user, _ := model.(*testUser)
			user.BirthTime = birthTime
			return nil
		}),
		def.BeforeCreate(func(model interface{}) error {
			user, _ := model.(*testUser)
			user.Age = int32(user.Now.Sub(user.BirthTime).Hours() / (24 * 365))
			return nil
		}),
		def.AfterCreate(func(model interface{}) error {
			user, _ := model.(*testUser)
			user.NickName = "nick name set by AfterCreate"
			return nil
		}),
	)

	// Test default factory
	user := &testUser{}
	err := Create(userFactory).To(user)
	if err != nil {
		t.Fatalf("Create failed with err=%s", err)
	}
	defer Delete(userFactory, user)

	checkUserForCreate(t, "Test Create default factory",
		&testUser{
			Name:     "test name",
			NickName: "nick name set by AfterCreate",
			Age:      17,
			Country:  "",
		},
		user,
	)
}

func TestCreateWithAssocitatoins(t *testing.T) {
	// define user factory
	userFactory := def.NewFactory(testUser{}, "test_user",
		def.Field("Name", "test name"),
		def.SequenceField("ID", 1, func(n int64) (interface{}, error) {
			return n, nil
		}),
	)
	// define blog factory
	blogFactory := def.NewFactory(testBlog{}, "test_blog",
		def.SequenceField("ID", 1, func(n int64) (interface{}, error) {
			return n, nil
		}),
		def.DynamicField("Title", func(blog interface{}) (interface{}, error) {
			blogInstance, _ := blog.(*testBlog)

			return fmt.Sprintf("Blog Title %d", blogInstance.ID), nil
		}),
		def.Association("Author", "AuthorID", "ID", userFactory,
			def.Field("Name", "blog author name"),
		),
	)

	// Test create with associtatoins
	blog := &testBlog{}
	err := Create(blogFactory,
		WithField("ID", int64(2)),
		WithField("Content", "Blog content2"),
	).To(blog)
	if err != nil {
		t.Fatalf("Create failed with error: %v", err)
	}
	defer Delete(userFactory, blog.Author)
	defer Delete(blogFactory, blog)

	checkBlogForCreate(t, "Test Create with association",
		&testBlog{
			ID:      2,
			Title:   "Blog Title 2",
			Content: "Blog content2",
		},
		blog,
	)
	checkUserForCreate(t, "Test Create with association",
		&testUser{
			Name: "blog author name",
		},
		blog.Author,
	)
}

func TestCreateOneToManyAssociation(t *testing.T) {
	// define blog factory
	blogFactory := def.NewFactory(testBlog{}, "test_blog",
		def.SequenceField("ID", 1, func(n int64) (interface{}, error) {
			return n, nil
		}),
		def.DynamicField("Title", func(blog interface{}) (interface{}, error) {
			blogInstance, ok := blog.(*testBlog)
			if !ok {
				return nil, fmt.Errorf("set field Title failed")
			}
			return fmt.Sprintf("Blog Title %d", blogInstance.ID), nil
		}),
	)
	// define user factory
	userFactory := def.NewFactory(testUser{}, "test_user",
		def.Field("Name", "test one-to-many name"),
		def.SequenceField("ID", 1, func(n int64) (interface{}, error) {
			return n, nil
		}),
		def.AfterCreate(func(user interface{}) error {
			author, _ := user.(*testUser)

			author.Blogs = []*testBlog{}
			return CreateSlice(blogFactory, 10,
				WithField("AuthorID", author.ID),
				WithField("Author", author),
			).To(&author.Blogs)
		}),
	)

	// Test Create one-to-many association
	user := &testUser{}
	err := Create(userFactory).To(user)
	if err != nil {
		t.Fatalf("Create failed with error: %v", err)
	}
	defer Delete(userFactory, user)
	for _, blog := range user.Blogs {
		defer Delete(blogFactory, blog)
	}

	if len(user.Blogs) != 10 {
		t.Fatalf("Create one-to-many association failed with len(Blogs)=%d, want len(Blogs)=10", len(user.Blogs))
	}
	for i, blog := range user.Blogs {
		checkBlogForCreate(t, "Test Create one-to-many association",
			&testBlog{
				ID:       int64(i) + 1,
				Title:    fmt.Sprintf("Blog Title %d", i+1),
				AuthorID: user.ID,
				Author: &testUser{
					ID:   user.ID,
					Name: "test one-to-many name",
				},
			},
			blog,
		)
	}
}

func TestCreateWithTraitContainsAssocitatoins(t *testing.T) {
	// define user factory
	userFactory := def.NewFactory(testUser{}, "test_user",
		def.Field("Name", "test name"),
		def.SequenceField("ID", 1, func(n int64) (interface{}, error) {
			return n, nil
		}),
	)
	// define blog factory
	blogFactory := def.NewFactory(testBlog{}, "test_blog",
		def.SequenceField("ID", 1, func(n int64) (interface{}, error) {
			return n, nil
		}),
		def.DynamicField("Title", func(blogIface interface{}) (interface{}, error) {
			blog, _ := blogIface.(*testBlog)

			return fmt.Sprintf("Blog Title %d", blog.ID), nil
		}),
		def.Association("Author", "AuthorID", "ID", userFactory,
			def.Field("Name", "blog author name"),
		),
	)
	// define star factory
	starFactory := def.NewFactory(testStar{}, "test_star",
		def.SequenceField("ID", 1, func(n int64) (interface{}, error) {
			return n, nil
		}),
		def.SequenceField("Count", 100, func(n int64) (interface{}, error) {
			return int(n), nil
		}),
		def.Trait("with blog",
			def.Association("Blog", "BlogID", "ID", blogFactory,
				def.Field("Title", "star blog title"),
			),
		),
	)

	// Test create without trait
	star := &testStar{}
	err := Create(starFactory).To(star)
	if err != nil {
		t.Fatalf("Create failed with error: %v", err)
	}
	defer Delete(starFactory, star)

	if star.Blog != nil {
		t.Errorf("Create with trait contains associations failed with star.Blog != nil")
	}

	// test with trait
	star = &testStar{}
	err = Create(starFactory, WithTraits("with blog")).To(star)
	if err != nil {
		t.Fatalf("Create failed with error: %v", err)
	}
	defer Delete(userFactory, star.Blog.Author)
	defer Delete(blogFactory, star.Blog)
	defer Delete(starFactory, star)

	checkBlogForCreate(t, "Test Create with association",
		&testBlog{
			ID:    star.BlogID,
			Title: "star blog title",
		},
		star.Blog,
	)
	checkUserForCreate(t, "Test Create with association",
		&testUser{
			Name: "blog author name",
		},
		star.Blog.Author,
	)
}

func TestCreateWithMultirelationAssocitatoins(t *testing.T) {
	// define user factory
	userFactory := def.NewFactory(testUser{}, "test_user",
		def.Field("Name", "test name"),
		def.SequenceField("ID", 1, func(n int64) (interface{}, error) {
			return n, nil
		}),
	)
	// define blog factory
	blogFactory := def.NewFactory(testBlog{}, "test_blog",
		def.SequenceField("ID", 1, func(n int64) (interface{}, error) {
			return n, nil
		}),
		def.DynamicField("Title", func(blogIface interface{}) (interface{}, error) {
			blog, _ := blogIface.(*testBlog)

			return fmt.Sprintf("Blog Title %d", blog.ID), nil
		}),
		def.Association("Author", "AuthorID", "ID", userFactory,
			def.Field("Name", "blog author name"),
		),
	)
	// define comment factory
	commentFactory := def.NewFactory(testComment{}, "test_comment",
		def.SequenceField("ID", 1, func(n int64) (interface{}, error) {
			return n, nil
		}),
		def.DynamicField("Text", func(comment interface{}) (interface{}, error) {
			commentInstance, ok := comment.(*testComment)
			if !ok {
				return nil, fmt.Errorf("set field Text failed")
			}
			return fmt.Sprintf("Comment Text %d", commentInstance.ID), nil
		}),
		def.Association("Blog", "BlogID", "ID", blogFactory,
			def.Field("Title", "comment blog title"),
		),
		def.Association("User", "UserID", "ID", userFactory,
			def.Field("Name", "comment user name"),
		),
	)

	// Test create with multirelation associtatoins
	comment := &testComment{}
	err := Create(commentFactory, WithField("ID", int64(10))).To(comment)
	if err != nil {
		t.Fatalf("Create failed with error: %v", err)
	}
	defer Delete(userFactory, comment.Blog.Author)
	defer Delete(blogFactory, comment.Blog)
	defer Delete(userFactory, comment.User)
	defer Delete(commentFactory, comment)

	// check comment
	checkCommentForCreate(t, "Test create with multirelation associtatoins",
		&testComment{
			ID:   10,
			Text: "Comment Text 10",
		},
		comment,
	)

	// check blog which comment belongs to
	checkBlogForCreate(t, "Test create with multirelation associtatoins",
		&testBlog{
			ID:      comment.BlogID,
			Title:   "comment blog title",
			Content: "",
		},
		comment.Blog,
	)
	// check author of comment blog
	checkUserForCreate(t, "Test Create with association",
		&testUser{
			ID:   comment.Blog.AuthorID,
			Name: "blog author name",
		},
		comment.Blog.Author,
	)
	// check comment user
	checkUserForCreate(t, "Test Create with association",
		&testUser{
			ID:   comment.UserID,
			Name: "comment user name",
		},
		comment.User,
	)
}

func TestCreateSlice(t *testing.T) {
	// define factory
	userFactory := def.NewFactory(testUser{}, "test_user",
		def.Field("Name", "test create slice name"),
		def.SequenceField("ID", 1, func(n int64) (interface{}, error) {
			return n, nil
		}),
	)

	// Test CreateSlice []*Type slice
	users := []*testUser{}
	err := CreateSlice(userFactory, 3).To(&users)
	if err != nil {
		t.Fatalf("CreateSlice failed with err=%s", err)
	}
	for _, user := range users {
		defer Delete(userFactory, user)
	}

	if len(users) != 3 {
		t.Fatalf("CreateSlice failed with len(users)=%d, want len(users)=3", len(users))
	}

	for i, user := range users {
		checkUserForCreate(t, "Test CreateSlice []*Type slice",
			&testUser{
				ID:   int64(i) + 1,
				Name: "test create slice name",
			},
			user,
		)
	}

	// Test CreateSlice []Type slice
	users2 := []testUser{}
	err = CreateSlice(userFactory, 3).To(&users2)
	if err != nil {
		t.Fatalf("CreateSlice failed with err=%s", err)
	}
	for _, user := range users2 {
		defer Delete(userFactory, user)
	}

	if len(users2) != 3 {
		t.Fatalf("CreateSlice failed with len(users2)=%d, want len(users2)=3", len(users2))
	}

	for i, user := range users2 {
		checkUserForCreate(t, "Test CreateSlice []Type slice",
			&testUser{
				ID:   int64(i) + 4,
				Name: "test create slice name",
			},
			&user,
		)
	}
}

func TestDelete(t *testing.T) {
	// define factory
	userFactory := def.NewFactory(testUser{}, "test_user",
		def.Field("Name", "test name"),
		def.SequenceField("ID", 1, func(n int64) (interface{}, error) {
			return n, nil
		}),
	)

	// Test default factory
	user := &testUser{}
	err := Create(userFactory).To(user)
	if err != nil {
		t.Fatalf("Create failed with err=%s", err)
	}

	err = Delete(userFactory, user)
	if err != nil {
		t.Fatalf("Delete failed with err=%v", err)
	}
}

func checkUserForCreate(t *testing.T, name string, expect *testUser, got *testUser) {
	if got.Name != expect.Name {
		t.Errorf("Case %s: failed with Name=%s, want Name=%s", name, got.Name, expect.Name)
	}
	if got.NickName != expect.NickName {
		t.Errorf("Case %s: failed with NickName=%s, want NickName=%s", name, got.NickName, expect.NickName)
	}
	if got.Age != expect.Age {
		t.Errorf("Case %s: failed with Age=%d, want Age=%d", name, got.Age, expect.Age)
	}
	if got.Country != expect.Country {
		t.Errorf("Case %s: failed with Country=%s, want Country=%s", name, got.Country, expect.Country)
	}
}

func checkBlogForCreate(t *testing.T, name string, expect *testBlog, got *testBlog) {
	if got.ID != expect.ID {
		t.Errorf("Case %s: failed with ID=%d, want ID=%d", name, got.ID, expect.ID)
	}
	if got.Title != expect.Title {
		t.Errorf("Case %s: failed with Title=%s, want Title=%s", name, got.Title, expect.Title)
	}
	if got.Content != expect.Content {
		t.Errorf("Case %s: failed with Content=%s, want Content=%s", name, got.Content, expect.Content)
	}
	if got.AuthorID != got.Author.ID {
		t.Errorf("Case %s: failed with AuthorID=%d, want AuthorID=%d", name, got.AuthorID, expect.AuthorID)
	}
}

func checkCommentForCreate(t *testing.T, name string, expect *testComment, got *testComment) {
	if got.Text != expect.Text {
		t.Errorf("Case %s: failed with Text=%s, want Text=%s", name, got.Text, expect.Text)
	}
	if got.BlogID != got.Blog.ID {
		t.Errorf("Case %s: failed with BlogID=%d, want BlogID=%d", name, got.BlogID, got.Blog.ID)
	}
	if got.UserID != got.User.ID {
		t.Errorf("Case %s: failed with UserID=%d, want UserID=%d", name, got.UserID, got.User.ID)
	}
}
