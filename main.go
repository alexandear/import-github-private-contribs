package main

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/big"
	"sort"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

const dateFormat = time.DateOnly

func main() {
	var (
		token          = flag.String("token", "", "GitHub access token")
		login          = flag.String("login", "", "GitHub login")
		fromDate       = flag.String("fromDate", "2021-01-01", fmt.Sprintf("Starting date in format %q", dateFormat))
		toDate         = flag.String("toDate", "2021-12-31", fmt.Sprintf("Ending date in format %q", dateFormat))
		committerName  = flag.String("committerName", "", `Commit author name in format "Name Surname"`)
		committerEmail = flag.String("committerEmail", "", `Commit author email in format "email@example.com"`)
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
	if *committerName == "" {
		log.Fatal("flag committerName must be non-empty")
	}
	if *committerEmail == "" {
		log.Fatal("flag committerEmail must be non-empty")
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

	var dayContributions []contribution
	for _, week := range contributionCalendar.Weeks {
		for _, day := range week.ContributionDays {
			if day.ContributionCount == 0 {
				continue
			}

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

	repoPath := "./repo." + *login

	repo, err := git.PlainInit(repoPath, false)
	if err == nil {
		log.Printf("Init repository %q", repoPath)
	} else if errors.Is(err, git.ErrRepositoryAlreadyExists) {
		repo, err = git.PlainOpen(repoPath)
		if err != nil {
			log.Fatalf("Failed to open repository %q: %v", repoPath, err)
		}

		log.Printf("Opened repository %q", repoPath)
	} else {
		log.Fatalf("Failed to init repository %q: %v", repoPath, err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		log.Fatalf("Failed to get worktree: %v", err)
	}

	for _, contrib := range dayContributions {
		if err := doCommits(worktree, contrib, *committerName, *committerEmail); err != nil {
			log.Printf("Failed to make commits for day %v: %v", contrib.Date, err)
			continue
		}
	}
}

type contribution struct {
	Count int
	Date  time.Time
}

func doCommits(wt *git.Worktree, contrib contribution, committerName, committerEmail string) error {
	sign := &object.Signature{
		Name:  committerName,
		Email: committerEmail,
	}

	whens := make([]time.Time, 0, contrib.Count)
	for i := 0; i < contrib.Count; i++ {
		whens = append(whens, randomWorkingTime(contrib.Date))
	}
	sort.Slice(whens, func(i, j int) bool {
		return whens[i].Before(whens[j])
	})

	for i := 0; i < contrib.Count; i++ {
		sign.When = whens[i]

		if _, err := wt.Commit("Private contribution", &git.CommitOptions{
			AllowEmptyCommits: true,
			Author:            sign,
			Committer:         sign,
		}); err != nil {
			return fmt.Errorf("commit: %w", err)
		}

		log.Printf("Committed %d time per day at %v", i+1, sign.When)
	}

	return nil
}

func randomWorkingTime(date time.Time) time.Time {
	hour := 8
	if h, err := rand.Int(rand.Reader, big.NewInt(12)); err == nil {
		hour = 8 + int(h.Int64())
	} else {
		log.Printf("Failed to generate rand hour: %v", err)
	}

	minute := 0
	if m, err := rand.Int(rand.Reader, big.NewInt(60)); err == nil {
		minute = int(m.Int64())
	} else {
		log.Printf("Failed to generate rand minute: %v", err)
	}

	second := 0
	if s, err := rand.Int(rand.Reader, big.NewInt(60)); err == nil {
		second = int(s.Int64())
	} else {
		log.Printf("Failed to generate rand second: %v", err)
	}

	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, second, 0, time.UTC)
}
