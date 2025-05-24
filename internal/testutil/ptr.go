package testutil

// StringPtr は文字列リテラルを string ポインタに変換するヘルパー関数
func StringPtr(s string) *string {
	return &s
}
