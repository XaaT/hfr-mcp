package hfr

// Post represents a single forum post
type Post struct {
	Numreponse int
	Author     string
	Date       string
	Content    string
}

// Topic holds metadata about a forum topic page
type Topic struct {
	Cat        int
	Post       int
	Page       int
	TotalPages int
	Posts      []Post
}

// EditInfo holds info parsed from an edit page
type EditInfo struct {
	IsFirstPost bool
	Subcat      string
	Subject     string
}
