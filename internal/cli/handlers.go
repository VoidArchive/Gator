package cli

import (
	"context"
	"database/sql"
	"fmt"
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
	feed, err := fetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		return fmt.Errorf("error fetching feed: %v", err)
	}
	fmt.Printf("Feed fetched successfully!\n")
	fmt.Printf("Channel Title: %s\n", feed.Channel.Title)
	fmt.Printf("Channel Link: %s\n", feed.Channel.Link)
	fmt.Printf("Channel Description: %s\n", feed.Channel.Description)
	fmt.Printf("Number of items: %d\n\n", len(feed.Channel.Item))

	for i, item := range feed.Channel.Item {
		fmt.Printf("Item %d:\n", i+1)
		fmt.Printf("  Title: %s\n", item.Title)
		fmt.Printf("  Link: %s\n", item.Link)
		fmt.Printf("  Description: %s\n", item.Description)
		fmt.Printf("  PubDate: %s\n\n", item.PubDate)
	}

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
