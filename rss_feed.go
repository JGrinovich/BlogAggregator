package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
)

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "gator")

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var result RSSFeed
	err = xml.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	result.Channel.Title = html.UnescapeString(result.Channel.Title)
	result.Channel.Description = html.UnescapeString(result.Channel.Description)

	for i := range result.Channel.Item {
		result.Channel.Item[i].Title = html.UnescapeString(result.Channel.Item[i].Title)
		result.Channel.Item[i].Description = html.UnescapeString(result.Channel.Item[i].Description)
	}

	return &result, nil
}

func scrapeFeeds(s *state, cmd command) error {
	ctx := context.Background()
	feed, err := s.db.GetNextFeedToFetch(ctx)
	if err != nil {
		return err
	}
	u_feed, err := s.db.MarkFeedFetched(ctx, feed.ID)
	if err != nil {
		return err
	}

	results, err := fetchFeed(ctx, u_feed.Url)
	if err != nil {
		return err
	}

	for _, result := range results.Channel.Item {
		fmt.Println("Title: ", result.Title)
		fmt.Println("URL: ", result.Link)
		fmt.Println("Date Published: ", result.PubDate)
	}
	return nil
}
