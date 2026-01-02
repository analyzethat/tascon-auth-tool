package models

import "time"

type User struct {
	PowerBIUserID int    `json:"id"`
	PowerBIUser   string `json:"email"`
}

type UserAccess struct {
	UserAccessID int       `json:"id"`
	UserID       int       `json:"userId"`
	GroupBkey    int       `json:"groupBkey"`
	GroupName    string    `json:"groupName"`
	CreationDate time.Time `json:"creationDate"`
}

type Group struct {
	GroupBkey int    `json:"groupBkey"`
	GroupName string `json:"groupName"`
}

type SearchResult struct {
	GroupBkey int    `json:"groupBkey"`
	GroupName string `json:"groupName"`
	MatchedOn string `json:"matchedOn"`
}
