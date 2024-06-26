package entity

import "time"

type Article struct {
	Id        uint64
	CreatedAt time.Time
	UpdatedAt time.Time
	//DeletedAt time.Time
	Author   Author
	Content  string
	Category string
	Title    string
	Status   uint8
}
type Author struct {
	Id   uint64
	Name string
}

type Interactive struct {
	BizId      int64
	ReadCnt    int64
	LikeCnt    int64
	CollectCnt int64
	Liked      bool
	Collected  bool
}

func (a Article) Abstract() string {
	str := []rune(a.Content)
	// 只取部分作为摘要
	if len(str) > 128 {
		str = str[:128]
	}
	return string(str)
}
