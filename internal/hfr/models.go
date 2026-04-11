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

// TopicListItem represents a topic in a category listing
type TopicListItem struct {
	PostID     int
	Title      string
	Author     string
	Replies    int
	Views      int
	LastPage   int
	LastDate   string
	LastAuthor string
	Sticky     bool
}

// TopicList holds results from a category listing page
type TopicList struct {
	Cat        int
	Subcat     int
	Page       int
	TotalPages int
	Topics     []TopicListItem
}

// EditInfo holds info parsed from an edit page
type EditInfo struct {
	IsFirstPost bool
	Subcat      string
	Subject     string
}
