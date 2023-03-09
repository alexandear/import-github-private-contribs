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

const dateFormat = time.DateOnly

func main() {
	var (
		token    = flag.String("token", "", "GitHub access token")
		login    = flag.String("login", "oredko-gd", "GitHub login")
		fromDate = flag.String("fromDate", "2021-01-01", fmt.Sprintf("Starting date in format %q", dateFormat))
		toDate   = flag.String("toDate", "2021-12-31", fmt.Sprintf("Ending date in format %q", dateFormat))
	)
	flag.Parse()

	if *token == "" {
		log.Fatal("flag token must be non-empty")
	}
	if *login == "" {
		log.Fatal("flag login must be non-empty")
	}
	from, err := time.Parse(dateFormat, *fromDate)
	if err != nil {
		log.Fatalf("flag fromDate must be correct: %v", err)
	}
	to, err := time.Parse(dateFormat, *toDate)
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

	contributionCalendar := queryContribution.User.ContributionsCollection.ContributionCalendar

	log.Printf("Total contributions for user %q between %v and %v: %d",
		*login, from, to, contributionCalendar.TotalContributions)

	type contribution struct {
		Count int
		Date  time.Time
	}

	var dayContributions []contribution
	for _, week := range contributionCalendar.Weeks {
		for _, day := range week.ContributionDays {
			contrib := contribution{
				Count: int(day.ContributionCount),
			}

			if date, err := time.Parse(time.DateOnly, string(day.Date)); err == nil {
				contrib.Date = date
			} else {
				log.Printf("Failed to parse date %q", string(day.Date))
			}

			dayContributions = append(dayContributions, contrib)
		}
	}

	log.Printf("Days contributed: %d", len(dayContributions))
}
