package redis

const (
	Prefix            = "bluebell:"
	KeyPostTimeZSet   = "post:time"
	KeyPostScoreZSet  = "post:score"
	KeyPostVoteZSetPF = "post:voted:"

	KeyCommunitySetPF = "community:"
)

func getRedisKey(key string) string {
	return Prefix + key
}
