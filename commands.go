package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/JGrinovich/BlogAggregator/internal/config"
	"github.com/JGrinovich/BlogAggregator/internal/database"
	"github.com/google/uuid"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	handlers map[string]func(*state, command) error
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("username required")
	}
	_, err := s.db.GetUser(context.Background(), cmd.args[0])
	if err == sql.ErrNoRows {
		fmt.Println("user does not exist")
		os.Exit(1)
	} else if err != nil {
		fmt.Println("user not found")
		os.Exit(1)
	}
	username := cmd.args[0]

	s.cfg.CurrentUserName = username

	if err := s.cfg.Write(); err != nil {
		return err
	}

	fmt.Println("user set to: ", username)
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("name required")
	}
	user, err := s.db.GetUser(context.Background(), cmd.args[0])
	if err == nil {
		fmt.Println("user already exists")
		os.Exit(1)
	} else if err == sql.ErrNoRows {
		user, err = s.db.CreateUser(context.Background(), database.CreateUserParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Name:      cmd.args[0],
		})
		if err != nil {
			return err
		}
		s.cfg.CurrentUserName = user.Name
		fmt.Println("user created")
		s.cfg.Write()
	} else {
		return err
	}
	return nil
}

func handlerReset(s *state, cmd command) error {
	err := s.db.DeleteUsers(context.Background())
	if err != nil {
		fmt.Println("reset unsuccessful")
		os.Exit(1)
	}
	fmt.Println("reset successful")
	return nil
}

func handlerList(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		fmt.Println("listing unsuccessful")
		os.Exit(1)
	}

	current := s.cfg.CurrentUserName

	for _, user := range users {
		if user.Name == current {
			fmt.Printf("* %s (current)\n", user.Name)
			continue
		}
		fmt.Printf("* %s\n", user.Name)
	}
	return nil
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("no arguments given")
	}
	duration, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return err
	}
	fmt.Println("Collecting feeds every ", duration)

	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	scrapeFeeds(s, cmd)

	for range ticker.C {
		scrapeFeeds(s, cmd)
	}
	return nil
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 2 {
		return fmt.Errorf("usage: %s <name> <url>", cmd.name)
	}

	name := cmd.args[0]
	url := cmd.args[1]
	ctx := context.Background()

	params := database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	}

	feed, err := s.db.CreateFeed(ctx, params)
	if err != nil {
		feed, err = s.db.GetFeedByURL(ctx, url)
		if err != nil {
			return err
		}
	}

	if _, err := s.db.CreateFeedFollow(ctx, database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	}); err != nil {
		return err
	}

	fmt.Println("ID:", feed.ID)
	fmt.Println("Name:", feed.Name)
	fmt.Println("URL:", feed.Url)
	fmt.Println("UserID:", feed.UserID)
	fmt.Println("CreatedAt:", feed.CreatedAt)
	fmt.Println("UpdatedAt:", feed.UpdatedAt)

	return nil
}

func handlerFeedsList(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("usage: %s", cmd.name)
	}

	feeds, err := s.db.ListFeedsWithUsers(context.Background())
	if err != nil {
		return fmt.Errorf("couldn't get feeds: %w", err)
	}

	for _, r := range feeds {
		fmt.Println("Feed Name:", r.FeedName)
		fmt.Println("Feed URL:", r.FeedUrl)
		fmt.Println("Username:", r.UserName)
		fmt.Println()
	}
	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("usage: follow <URL>")
	}

	url := cmd.args[0]
	ctx := context.Background()

	feed, err := s.db.GetFeedByURL(ctx, url)
	if err != nil {
		return err
	}

	ff, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("couldn't follow feed: %w", err)
	}
	fmt.Println("Feed Name:", ff.FeedName)
	fmt.Println("User's Name:", ff.UserName)
	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("usage: following")
	}

	ctx := context.Background()

	ff, err := s.db.GetFeedFollowsForUser(ctx, user.ID)
	if err != nil {
		return err
	}

	for _, feed := range ff {
		fmt.Println("Follows:", feed.FeedName)
	}

	return nil
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			return err
		}
		return handler(s, cmd, user)
	}
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("usage: unfollow <URL>")
	}

	url := cmd.args[0]
	ctx := context.Background()

	feed, err := s.db.GetFeedByURL(ctx, url)
	if err != nil {
		return err
	}

	return s.db.DeleteFeedFollowByUserAndFeed(ctx, database.DeleteFeedFollowByUserAndFeedParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})
}

func handlerBrowse(s *state, cmd command) error {
	if s.cfg.CurrentUserName == "" {
		return fmt.Errorf("no current user")
	}
	ctx := context.Background()
	user, err := s.db.GetUser(ctx, s.cfg.CurrentUserName)
	if err != nil {
		return err
	}

	lim := 2
	if len(cmd.args) > 0 {
		if v, err := strconv.Atoi(cmd.args[0]); err == nil && v > 0 {
			lim = v
		}
	}

	posts, err := s.db.GetPostsForUser(ctx, database.GetPostsForUserParams{UserID: user.ID, Limit: int32(lim)})
	if err != nil {
		return err
	}

	for _, p := range posts {
		t := p.CreatedAt
		if p.PublishedAt.Valid {
			t = p.PublishedAt.Time
		}
		fmt.Printf("%s (%s)\n%s\n%s\n\n", p.Title, p.FeedName, p.Url, t.Format(time.RFC3339))
	}

	return nil
}

func (c *commands) run(s *state, cmd command) error {
	h, ok := c.handlers[cmd.name]
	if !ok {
		return fmt.Errorf("unknown command: %s", cmd.name)
	}
	return h(s, cmd)
}

func (c *commands) register(name string, f func(*state, command) error) {
	if c.handlers == nil {
		c.handlers = make(map[string]func(*state, command) error)
	}
	c.handlers[name] = f
}
