package cli

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/voidarchive/Gator/internal/database"
)

func MiddlewareLoggedIn(handler func(s *State, cmd Command, user database.User) error) func(*State, Command) error {
	return func(s *State, cmd Command) error {
		user, err := s.DB.GetUser(context.Background(), s.Cfg.CurrentUserName)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("user %s doesn't exist", s.Cfg.CurrentUserName)
			}
			return fmt.Errorf("error getting user: %v", err)
		}
		return handler(s, cmd, user)
	}
}

func HandlerLogin(s *State, cmd Command) error {
	if len(cmd.Args) == 0 {
		return fmt.Errorf("login requires a username")
	}
	username := cmd.Args[0]
	if _, err := s.DB.GetUser(context.Background(), username); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user %s doesn't exist", username)
		}
		return fmt.Errorf("error checking user: %v", err)
	}

	if err := s.Cfg.SetUser(username); err != nil {
		return err
	}

	fmt.Printf("User has been set to: %s\n", username)
	return nil
}

func HandlerRegister(s *State, cmd Command) error {
	if len(cmd.Args) == 0 {
		return fmt.Errorf("register requires a username")
	}
	username := cmd.Args[0]

	user, err := s.DB.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      username,
	})
	if err != nil {
		return fmt.Errorf("error creating user: %v", err)
	}
	if err := s.Cfg.SetUser(username); err != nil {
		return fmt.Errorf("error setting current user: %v", err)
	}

	fmt.Printf("User %s created successfully!\n", username)
	fmt.Printf("User data: ID=%s, Name=%s, CreatedAt=%s\n", user.ID, user.Name, user.CreatedAt.Format(time.RFC3339))
	return nil
}

func HandlerReset(s *State, cmd Command) error {
	if err := s.DB.DeleteAllUsers(context.Background()); err != nil {
		return fmt.Errorf("error resetting database :%v", err)
	}

	fmt.Println("Database reset successfully")
	return nil
}

func HandlerUsers(s *State, cmd Command) error {
	users, err := s.DB.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("error getting users: %v", err)
	}
	currentUser := s.Cfg.CurrentUserName

	for _, user := range users {
		if user.Name == currentUser {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}
	return nil
}

func HandlerAgg(s *State, cmd Command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("usage: agg <time_between_reqs>")
	}

	timeBetweenReqs, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		return fmt.Errorf("invalid duration: %v", err)
	}

	fmt.Printf("Collecting feeds every %v\n", timeBetweenReqs)

	ticker := time.NewTicker(timeBetweenReqs)
	defer ticker.Stop()

	// Run immediately first, then wait for ticker
	for ; ; <-ticker.C {
		err := scrapeFeeds(s)
		if err != nil {
			fmt.Printf("Error scraping feeds: %v\n", err)
			// Continue the loop even if there's an error
		}
	}
}

func parseRSSTime(timeStr string) (time.Time, error) {
	// Common RSS time formats
	formats := []string{
		time.RFC1123Z,               // "Mon, 02 Jan 2006 15:04:05 -0700"
		time.RFC1123,                // "Mon, 02 Jan 2006 15:04:05 MST"
		time.RFC822Z,                // "02 Jan 06 15:04 -0700"
		time.RFC822,                 // "02 Jan 06 15:04 MST"
		"2006-01-02T15:04:05Z07:00", // ISO 8601
		"2006-01-02 15:04:05",       // Simple format
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", timeStr)
}

func scrapeFeeds(s *State) error {
	ctx := context.Background()

	// Get the next feed to fetch
	feed, err := s.DB.GetNextFeedToFetch(ctx)
	if err != nil {
		return fmt.Errorf("error getting next feed to fetch: %v", err)
	}

	fmt.Printf("Fetching feed: %s (%s)\n", feed.Name, feed.Url)

	// Mark the feed as fetched before actually fetching to avoid race conditions
	err = s.DB.MarkFeedFetched(ctx, feed.ID)
	if err != nil {
		return fmt.Errorf("error marking feed as fetched: %v", err)
	}

	// Fetch the RSS feed
	rssFeed, err := fetchFeed(ctx, feed.Url)
	if err != nil {
		return fmt.Errorf("error fetching RSS feed %s: %v", feed.Url, err)
	}

	// Save posts to database
	fmt.Printf("Found %d posts from %s:\n", len(rssFeed.Channel.Item), rssFeed.Channel.Title)
	for _, item := range rssFeed.Channel.Item {
		// Parse the published date - handle different formats
		var publishedAt sql.NullTime
		if item.PubDate != "" {
			// Try common RSS date formats
			parsedTime, err := parseRSSTime(item.PubDate)
			if err == nil {
				publishedAt = sql.NullTime{Time: parsedTime, Valid: true}
			}
		}

		// Create post in database
		_, err := s.DB.CreatePost(ctx, database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Title:       item.Title,
			Url:         item.Link,
			Description: sql.NullString{String: item.Description, Valid: item.Description != ""},
			PublishedAt: publishedAt,
			FeedID:      feed.ID,
		})
		if err != nil {
			// If it's a duplicate URL error, just skip it
			if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
				continue
			}
			// Log other errors but continue processing
			fmt.Printf("Error saving post %s: %v\n", item.Title, err)
		}
	}
	fmt.Printf("Processed %d posts from %s\n\n", len(rssFeed.Channel.Item), rssFeed.Channel.Title)

	return nil
}

func HandlerAddFeed(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) < 2 {
		return fmt.Errorf("addFeed requres name and url arguments")
	}
	name := cmd.Args[0]
	url := cmd.Args[1]

	feed, err := s.DB.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	})

	if err != nil {
		return fmt.Errorf("error creating feed: %v", err)
	}

	// Automatically follow the feed that was just created
	_, err = s.DB.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("error following feed: %v", err)
	}

	fmt.Printf("Feed added successfully!\n")
	fmt.Printf("Feed data: ID=%s, Name=%s, URL=%s, UserID=%s, CreatedAt=%s\n", feed.ID, feed.Name, feed.Url, user.ID, feed.CreatedAt.Format(time.RFC3339))
	return nil
}

func HandlerListFeeds(s *State, cmd Command) error {
	feeds, err := s.DB.ListAllFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("error listing feeds: %v", err)
	}
	fmt.Printf("Feeds:\n")
	for _, feed := range feeds {
		fmt.Printf("  Name: %s\n", feed.FeedName)
		fmt.Printf("  URL: %s\n", feed.Url)
		fmt.Printf("  Created by: %s\n\n", feed.UserName)
	}
	return nil
}

func HandlerFollow(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("usage: follow <feed-url")
	}
	url := cmd.Args[0]
	ctx := context.Background()

	feed, err := s.DB.GetFeedByUrl(ctx, url)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("feed not found")
		}
		return fmt.Errorf("error getting feed: %v", err)
	}

	follow, err := s.DB.CreateFeedFollow(ctx, database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("error creating feed follow: %v", err)
	}
	fmt.Printf("User %s is now following feed %s\n", follow.UserName, follow.FeedName)

	return nil
}

func HandlerFollowing(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) != 0 {
		return fmt.Errorf("usage: following")
	}
	ctx := context.Background()

	follows, err := s.DB.GetFeedFollowsForUser(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("error fetching follows: %v", err)
	}

	fmt.Println("Following feeds:")
	for _, f := range follows {
		fmt.Printf("* %s\n", f.FeedName)
	}
	return nil
}

func HandlerUnfollow(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("usage: unfollow <feed-url>")
	}
	url := cmd.Args[0]
	ctx := context.Background()

	// First check if the feed exists
	feed, err := s.DB.GetFeedByUrl(ctx, url)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("feed not found")
		}
		return fmt.Errorf("error getting feed: %v", err)
	}

	// Delete the feed follow record
	err = s.DB.DeleteFeedFollow(ctx, database.DeleteFeedFollowParams{
		UserID: user.ID,
		Url:    url,
	})
	if err != nil {
		return fmt.Errorf("error unfollowing feed: %v", err)
	}

	fmt.Printf("User %s has unfollowed feed %s\n", user.Name, feed.Name)
	return nil
}

func HandlerBrowse(s *State, cmd Command, user database.User) error {
	limit := 2 // Default limit
	if len(cmd.Args) > 0 {
		parsedLimit, err := strconv.Atoi(cmd.Args[0])
		if err != nil {
			return fmt.Errorf("invalid limit: %v", err)
		}
		if parsedLimit <= 0 {
			return fmt.Errorf("limit must be positive")
		}
		limit = parsedLimit
	}

	ctx := context.Background()
	posts, err := s.DB.GetPostsForUser(ctx, database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		return fmt.Errorf("error getting posts: %v", err)
	}

	if len(posts) == 0 {
		fmt.Println("No posts found from your followed feeds")
		return nil
	}

	fmt.Printf("Recent posts from your followed feeds (showing %d):\n\n", len(posts))
	for _, post := range posts {
		fmt.Printf("Title: %s\n", post.Title)
		fmt.Printf("Feed: %s\n", post.FeedName)
		if post.Description.Valid && post.Description.String != "" {
			fmt.Printf("Description: %s\n", post.Description.String)
		}
		fmt.Printf("URL: %s\n", post.Url)
		if post.PublishedAt.Valid {
			fmt.Printf("Published: %s\n", post.PublishedAt.Time.Format("January 2, 2006 at 3:04 PM"))
		}
		fmt.Println("=====================================")
	}
	return nil
}
