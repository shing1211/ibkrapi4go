// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Program seed_discussions seeds the GitHub Discussions forum with starter
// threads. Run with a GitHub personal access token in GH_TOKEN:
//
//	GH_TOKEN=$(gh auth token) go run scripts/seed_discussions.go
//
// The token needs the "discussions" scope. Public repos need public_repo scope.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	owner  = "shing1211"
	repo    = "ibkrapi4go"
	gateway = "https://api.github.com/graphql"
)

func main() {
	token := os.Getenv("GH_TOKEN")
	if token == "" {
		token = os.Getenv("GH_AUTH_TOKEN")
	}
	if token == "" {
		fmt.Println("Error: GH_TOKEN environment variable not set")
		fmt.Println("Run: GH_TOKEN=$(gh auth token) go run scripts/seed_discussions.go")
		os.Exit(1)
	}

	c := &http.Client{Timeout: 30 * time.Second}
	ctx := context.Background()

	repoID, err := getRepoID(ctx, c, token)
	if err != nil {
		fmt.Printf("Error getting repo ID: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Repo ID: %s\n", repoID)

	categories, err := getCategories(ctx, c, token, repoID)
	if err != nil {
		fmt.Printf("Error getting categories: %v\n", err)
		os.Exit(1)
	}

	catByName := func(name string) string {
		for _, c := range categories {
			if c.Name == name {
				return c.ID
			}
		}
		return ""
	}

	type discussion struct{ title, body, categoryName string }
	seeds := []discussion{
		{
			title:       "Welcome to ibkrapi4go Discussions!",
			body:        "Hello! This is the official discussion forum for the ibkrapi4go Go SDK for Interactive Brokers.\n\nFeel free to ask questions about using the SDK, share what you're building, or discuss feature requests. Please keep discussions respectful and on-topic.\n\n**Getting started:**\n- Read the [README](https://github.com/shing1211/ibkrapi4go#readme)\n- Check the [API reference](https://github.com/shing1211/ibkrapi4go/tree/main/docs)\n- Report bugs via [GitHub Issues](https://github.com/shing1211/ibkrapi4go/issues)\n\nThis forum is maintained on a best-effort basis by volunteers.",
			categoryName: "General",
		},
		{
			title:       "How do I get started with paper trading?",
			body:        "This is a placeholder for a frequently-asked question about getting started with paper trading credentials.\n\n**Answer:** To use paper trading with ibkrapi4go, you need:\n1. An IBKR paper trading account (create one at ibkr.com)\n2. The Client Portal Gateway running locally\n3. Set environment variables: IBKR_GATEWAY, IBKR_USERNAME, IBKR_PASSWORD\n\nSee the [live examples](https://github.com/shing1211/ibkrapi4go/tree/main/examples/live-portfolio) for a working reference.",
			categoryName: "Q&A",
		},
		{
			title:       "Feature requests and roadmap suggestions",
			body:        "This is a placeholder for collecting feature requests and roadmap suggestions for the v1.0 release.\n\nKnown items being considered:\n- **P3**: Real-world example suite (examples against live paper trading) — in progress\n- **P5**: v1.0.0 final stabilization and changelog sweep\n- **P6**: Multi-gateway / session pool for aggregate multi-account views\n\nPlease comment below with any features you'd like to see!",
			categoryName: "Ideas",
		},
	}

	for _, seed := range seeds {
		catID := catByName(seed.categoryName)
		if catID == "" {
			fmt.Printf("Category %q not found, skipping\n", seed.categoryName)
			continue
		}

		id, err := createDiscussion(ctx, c, token, repoID, catID, seed.title, seed.body)
		if err != nil {
			fmt.Printf("Error creating discussion %q: %v\n", seed.title, err)
			continue
		}
		fmt.Printf("Created discussion %q (id=%s, category=%s)\n", seed.title, id, seed.categoryName)
	}

	fmt.Println("\nDone! Check https://github.com/shing1211/ibkrapi4go/discussions")
}

type gqlResponse struct {
	Data   json.RawMessage
	Errors []struct{ Message string }
}

func doGraphQL(ctx context.Context, c *http.Client, token, query string, variables map[string]any) (json.RawMessage, error) {
	payload := map[string]any{"query": query}
	if variables != nil {
		payload["variables"] = variables
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", gateway, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	req.Header.Set("Authorization", "bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	var r gqlResponse
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if len(r.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL errors: %v", r.Errors)
	}
	return r.Data, nil
}

func getRepoID(ctx context.Context, c *http.Client, token string) (string, error) {
	query := `query($owner:String!$name:String!){ repository(owner:$owner,name:$name){ id } }`
	vars := map[string]any{"owner": owner, "name": repo}
	data, err := doGraphQL(ctx, c, token, query, vars)
	if err != nil {
		return "", err
	}
	var v struct{ Repository struct{ ID string } }
	if err := json.Unmarshal(data, &v); err != nil {
		return "", fmt.Errorf("unmarshal: %w", err)
	}
	return v.Repository.ID, nil
}

func getCategories(ctx context.Context, c *http.Client, token, repoID string) ([]struct{ Name, ID string }, error) {
	query := `query($id:ID!){ node(id:$id){ ... on Repository { discussionCategories(first:10) { nodes { name id } } } } }`
	vars := map[string]any{"id": repoID}
	data, err := doGraphQL(ctx, c, token, query, vars)
	if err != nil {
		return nil, err
	}
	var v struct {
		Node struct {
			DiscussionCategories struct {
				Nodes []struct{ Name, ID string }
			}
		}
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return v.Node.DiscussionCategories.Nodes, nil
}

func createDiscussion(ctx context.Context, c *http.Client, token, repoID, categoryID, title, body string) (string, error) {
	mutation := `mutation($repo:ID!$cat:ID!$title:String!$body:String!){
		createDiscussion(input:{repositoryId:$repo,categoryId:$cat,title:$title,body:$body}){
			discussion{ id }
		}
	}`
	vars := map[string]any{
		"repo":   repoID,
		"cat":    categoryID,
		"title":  title,
		"body":   body,
	}
	data, err := doGraphQL(ctx, c, token, mutation, vars)
	if err != nil {
		return "", err
	}
	var v struct {
		CreateDiscussion struct {
			Discussion struct{ ID string }
		}
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return "", fmt.Errorf("unmarshal: %w", err)
	}
	return v.CreateDiscussion.Discussion.ID, nil
}
