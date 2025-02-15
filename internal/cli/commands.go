package cli

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gator/internal/aggregator"
	"gator/internal/config"
	"gator/internal/database"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type State struct {
	Config *config.Config
	DB     *database.Queries
}

type Command struct {
	Name string
	Args []string
}

type HandlerFunc func(s *State, cmd Command) error

type Commands struct {
	handlers map[string]HandlerFunc
}

func NewCommands() *Commands {
	return &Commands{
		handlers: make(map[string]HandlerFunc),
	}
}

func (c *Commands) Register(name string, f HandlerFunc) {
	c.handlers[name] = f
}

func (c *Commands) Run(s *State, cmd Command) error {
	handler, exists := c.handlers[cmd.Name]
	if !exists {
		return fmt.Errorf("unknown comand: %s", cmd.Name)
	}
	return handler(s, cmd)
}

func HandlerRegister(s *State, cmd Command) error {
	if len(cmd.Args) < 1 {
		return errors.New("username is required")
	}
	name := cmd.Args[0]

	_, err := s.DB.GetUser(context.Background(), name)
	if err == nil {
		return fmt.Errorf("user %s already exists", name)
	} else if err != sql.ErrNoRows {
		return err
	}

	now := time.Now()
	newUser, err := s.DB.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Name:      name,
	})
	if err != nil {
		return err
	}

	if err := s.Config.SetUser(name); err != nil {
		return err
	}

	fmt.Printf("User %s create successfully!\nUserData: %+v\n", name, newUser)
	return nil
}

func MiddlewareLoggedIn(
	handler func(s *State, cmd Command, user database.User) error,
) func(s *State, cmd Command) error {
	return func(s *State, cmd Command) error {
		user, err := s.DB.GetUser(context.Background(), s.Config.CurrentUserName)
		if err != nil {
			return fmt.Errorf("failed to retrieve logged-in user: %w", err)
		}
		return handler(s, cmd, user)
	}
}

func HandlerLogin(s *State, cmd Command) error {
	if len(cmd.Args) < 1 {
		return errors.New("username is required")
	}
	username := cmd.Args[0]

	user, err := s.DB.GetUser(context.Background(), username)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user %s does not exist", username)
		}
		return err
	}

	if err := s.Config.SetUser(username); err != nil {
		return err
	}

	fmt.Printf("Logged in as %s. User data: %+v\n", username, user)
	return nil
}

func HandlerReset(s *State, cmd Command) error {
	err := s.DB.ResetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed to reset database: %w", err)
	}
	fmt.Println("Database reset successfully!")
	return nil
}

func HandlerUsers(s *State, cmd Command) error {
	users, err := s.DB.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	current := s.Config.CurrentUserName

	for _, u := range users {
		if u.Name == current {
			fmt.Printf("* %s (current)\n", u.Name)
		} else {
			fmt.Printf("* %s\n", u.Name)
		}
	}
	return nil
}

func HandlerAgg(s *State, cmd Command) error {
	if len(cmd.Args) < 1 {
		return errors.New("time_between_reqs is required (e.g., 1m)")
	}
	durationStr := cmd.Args[0]
	timeBetweenReqs, err := time.ParseDuration(durationStr)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	fmt.Printf("Collecting feeds every %s\n", timeBetweenReqs)
	ticker := time.NewTicker(timeBetweenReqs)
	defer ticker.Stop()

	for {
		if err := scrapeFeeds(s); err != nil {
			fmt.Printf("Error scraping feeds: %v\n", err)
		}
		<-ticker.C
	}
}

func HandlerAddFeedLogged(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) < 2 {
		return errors.New("feed name and URL are required")
	}

	feedName := cmd.Args[0]
	feedURL := cmd.Args[1]

	now := time.Now()
	newFeed, err := s.DB.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Name:      feedName,
		Url:       feedURL,
		UserID:    user.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to create feed: %w", err)
	}

	_, err = s.DB.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		UserID:    user.ID,
		FeedID:    newFeed.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to automatically follow feed: %w", err)
	}

	fmt.Printf("Feed created successfully: %+v\n", newFeed)
	return nil
}

func HandlerFeeds(s *State, cmd Command) error {
	feeds, err := s.DB.GetAllFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get feeds: %w", err)
	}

	if len(feeds) == 0 {
		fmt.Println("No feeds found.")
		return nil
	}

	for _, feed := range feeds {
		fmt.Printf("Feed: %s\nURL: %s\nCreated by: %s\n\n", feed.FeedName, feed.Url, feed.UserName)
	}
	return nil
}

func HandlerFollowLogged(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) < 1 {
		return errors.New("feed URL is required")
	}
	feedURL := cmd.Args[0]

	feed, err := s.DB.GetFeedByURL(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("failed to find feed with URL %s: %w", feedURL, err)
	}

	now := time.Now()
	follow, err := s.DB.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to follow feed: %w", err)
	}

	fmt.Printf("User %s now follows feed %s\n", follow.UserName, follow.FeedName)
	return nil
}

func HandlerFollowingLogged(s *State, cmd Command, user database.User) error {
	follows, err := s.DB.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("failed to get feed follows: %w", err)
	}

	if len(follows) == 0 {
		fmt.Println("No feed follows found.")
		return nil
	}

	fmt.Println("Following:")
	for _, ff := range follows {
		fmt.Printf("- %s\n", ff.FeedName)
	}
	return nil
}

func HandlerUnfollowLogged(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) < 1 {
		return errors.New("feed URL is required")
	}
	feedURL := cmd.Args[0]

	err := s.DB.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{
		UserID: user.ID,
		Url:    feedURL,
	})
	if err != nil {
		return fmt.Errorf("failed to unfollow feed: %w", err)
	}
	fmt.Printf("User %s unfollowed feed with URL %s\n", user.Name, feedURL)
	return nil
}

func scrapeFeeds(s *State) error {
	ctx := context.Background()

	feed, err := s.DB.GetNextFeedToFetch(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No feed to fetch.")
			return nil
		}
		return fmt.Errorf("failed to get next feed: %w", err)
	}

	if err := s.DB.MarkedFeedFetched(ctx, feed.ID); err != nil {
		return fmt.Errorf("failed to mark feed fetched: %w", err)
	}

	rssFeed, err := aggregator.FetchFeed(ctx, feed.Url)
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %w", err)
	}

	now := time.Now()
	for _, item := range rssFeed.Channel.Item {
		var publishedAt time.Time
		if item.PubDate != "" {
			publishedAt, err = time.Parse(time.RFC1123, item.PubDate)
			if err != nil {
				publishedAt, err = time.Parse(time.RFC1123Z, item.PubDate)
				if err != nil {
					publishedAt = time.Time{}
				}
			}
		}

		err = s.DB.CreatePost(ctx, database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   now,
			UpdatedAt:   now,
			Title:       item.Title,
			Url:         item.Link,
			Description: sql.NullString{String: item.Description, Valid: item.Description != ""},
			PublishedAt: sql.NullTime{Time: publishedAt, Valid: !publishedAt.IsZero()},
			FeedID:      feed.ID,
		})
		if err != nil {
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {

			} else {
				fmt.Printf("Failed to create post for '%s': %v\n", item.Title, err)
			}
		}
	}
	return nil
}

func HandlerBrowsePostsLogged(s *State, cmd Command, user database.User) error {
	limit := 2
	if len(cmd.Args) >= 1 {
		if parsedLimit, err := strconv.Atoi(cmd.Args[0]); err == nil {
			limit = parsedLimit
		}
	}

	posts, err := s.DB.GetPostsForUSer(context.Background(), database.GetPostsForUSerParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		return fmt.Errorf("failed to get posts for user: %w", err)
	}

	if len(posts) == 0 {
		fmt.Println("No posts found.")
		return nil
	}

	for _, post := range posts {
		publishedAt := "N/A"
		if post.PublishedAt.Valid {
			publishedAt = post.PublishedAt.Time.Format(time.RFC3339)
		}
		fmt.Printf("Title: %s\nURL: %s\n Published: %s\n\n", post.Title, post.Url, publishedAt)
	}
	return nil
}
