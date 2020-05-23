package twitter

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

type Client interface {
	Tweet(msg string) error
}

type client struct {
	cli *twitter.Client
}

func (c *client) Tweet(msg string) error {
	_, _, err := c.cli.Statuses.Update(msg, nil)
	return err
}

type noopClient struct {
}

func (n *noopClient) Tweet(msg string) error {
	return nil
}

func New(
	twitterConsumerKey string,
	twitterConsumerSecret string,
	twitterAccessToken string,
	twitterAccessTokenSecret string,
) Client {
	if twitterConsumerKey == "" {
		return &noopClient{}
	}

	config := oauth1.NewConfig(twitterConsumerKey, twitterConsumerSecret)
	token := oauth1.NewToken(twitterAccessToken, twitterAccessTokenSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	return &client{cli: twitter.NewClient(httpClient)}
}
