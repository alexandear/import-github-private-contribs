package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

func main() {
	var (
		token    = flag.String("token", "", "GitHub access token")
		login    = flag.String("login", "oredko-gd", "GitHub login")
		fromDate = flag.String("fromDate", "2021-01-01 00:00:00", fmt.Sprintf("Starting date in format %q", time.DateTime))
		toDate   = flag.String("toDate", "2021-12-31 00:00:00", fmt.Sprintf("Ending date in format %q", time.DateTime))
	)
	flag.Parse()

	if *token == "" {
		log.Fatal("flag token must be non-empty")
	}
	if *login == "" {
		log.Fatal("flag login must be non-empty")
	}
	from, err := time.Parse(time.DateTime, *fromDate)
	if err != nil {
		log.Fatalf("flag fromDate must be correct: %v", err)
	}
	to, err := time.Parse(time.DateTime, *toDate)
	if err != nil {
		log.Fatalf("flag toDate must be correct: %v", err)
	}

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)

	variables := map[string]any{
		"login": githubv4.String(*login),
		"from": githubv4.DateTime{
			Time: from,
		},
		"to": githubv4.DateTime{
			Time: to,
		},
	}

	var queryContribution struct {
		User struct {
			ContributionsCollection struct {
				ContributionCalendar struct {
					Weeks []struct {
						ContributionDays []struct {
							ContributionCount githubv4.Int
							Date              githubv4.String
						}
					}
					TotalContributions githubv4.Int
				}
			} `graphql:"contributionsCollection(from: $from, to: $to)"`
		} `graphql:"user(login: $login)"`
	}
	if err := client.Query(context.Background(), &queryContribution, variables); err != nil {
		log.Fatalf("Failed to get contributions: %v", err)
	}

	log.Printf("%+v", queryContribution.User.ContributionsCollection)
}
