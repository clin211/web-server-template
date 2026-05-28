package id

// NewCode 可以根据 id 生成唯一编码（你需要确保 id 是唯一的）。
func NewCode(id uint64, options ...func(*CodeOptions)) string {
	ops := getCodeOptionsOrSetDefault(nil)
	for _, f := range options {
		f(ops)
	}
	// 扩大并加盐
	id = id*uint64(ops.n1) + ops.salt

	var code []rune
	slIdx := make([]byte, ops.l)

	charLen := len(ops.chars)
	charLenUI := uint64(charLen)

	// 扩散
	for i := 0; i < ops.l; i++ {
		slIdx[i] = byte(id % charLenUI)                          // 获取每个数字
		slIdx[i] = (slIdx[i] + byte(i)*slIdx[0]) % byte(charLen) // 让个位数影响其他位数
		id /= charLenUI                                          // 右移
	}

	// 混淆(https://en.wikipedia.org/wiki/Permutation_box)
	for i := 0; i < ops.l; i++ {
		idx := (byte(i) * byte(ops.n2)) % byte(ops.l)
		code = append(code, ops.chars[slIdx[idx]])
	}
	return string(code)
}
