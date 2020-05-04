package expr


// Source state of filter scanner. We expect a predicate here. A predicate can start with an attribute path name, or
// a left parenthesis (for grouping), or the first character of the 'not' logical operator.
func (fs *filterScanner) stateBeginComplexPredicate(scan *filterScanner, c byte) int {
	if c == ' ' {
		return scanFilterSkipSpace
	}

	switch c {
	case 'n', 'N':
		// we are not sure whether the token would be 'not', or just a path starting with 'n'.
		scan.step = fs.stateN
		return scanFilterBeginAny
	case '(':
		fs.parenLevel++
		scan.step = fs.stateBeginComplexPredicate
		return scanFilterParenthesis
	}

	// A simple alphabet that does not start with 'n' or 'N', hence could not be 'not': this
	// should be a path
	if isFirstAlphabet(c) {
		scan.step = fs.stateInComplexPath
		return scanFilterContinue
	}

	return fs.error(c, "invalid character at the start of the predicate")
}

// Intermediate state where the predicate ends. A right parenthesis could signal the end of grouping; 'a' and 'o' could
// signal logical and/or operator; the termination byte could signal end of the filter.
func (fs *filterScanner) stateEndComplexPredicate(scan *filterScanner, c byte) int {
	if c == ' ' {
		return scanFilterSkipSpace
	}

	switch c {
	case ')':
		scan.parenLevel--
		return scanFilterParenthesis
	case 'a', 'A':
		// logical and
		scan.step = fs.stateComplexOpA
		return scanFilterContinue
	case 'o', 'O':
		// logical or
		scan.step = fs.stateComplexOpO
		return scanFilterContinue
	case 0:
		scan.step = fs.stateEOF
		return scanFilterEnd
	}

	return fs.error(c, "invalid character at the end of the predicate")
}

// Intermediate state where the last character was 'n' (case insensitive) at the start of the predicate. This could lead
// to a logical not operator if the current character is 'o' (case insensitive), or lead to a path name instead.
func (fs *filterScanner) stateComplexN(scan *filterScanner, c byte) int {
	switch c {
	case 'o', 'O':
		scan.step = fs.stateNo
		return scanFilterContinue
	}

	// just a path
	if c == '.' || c == ':' || isNonFirstAlphabet(c) {
		scan.step = fs.stateInComplexPath
		return scanFilterContinue
	}

	return fs.error(c, "invalid character in complex path")
}

// Intermediate state where the last two characters were 'n' and 'o' (case insensitive) at the start of the predicate.
// This could lead to a logical not operator if the current character is 't' (case insensitive), or lead to a path
// name instead.
func (fs *filterScanner) stateComplexNo(scan *filterScanner, c byte) int {
	switch c {
	case 't', 'T':
		scan.step = fs.stateNot
		return scanFilterContinue
	}

	// just a path
	if c == '.' || c == ':' || isNonFirstAlphabet(c) {
		scan.step = fs.stateInComplexPath
		return scanFilterContinue
	}

	return fs.error(c, "invalid character in complex path")
}

// Intermediate state where the last three characters were 'n', 'o' and 't' (case insensitive) at the start of the predicate.
// This could lead to a logical not operator if the current character is can indicate the end of an operator. Otherwise,
// this could only lead to a path name instead.
func (fs *filterScanner) stateComplexNot(scan *filterScanner, c byte) int {
	switch c {
	case ' ':
		scan.step = fs.stateBeginComplexPredicate
		return scanFilterContinue
	case '(':
		// ask caller to replay with a space so we can enter the condition above
		// the parenthesis count will be incremented after the replay so we don't
		// deal with it here
		return scanFilterInsertSpace
	}

	// seem like just a path that starts with 'not' (i.e. notes.title)
	if c == '.' || c == ':' || isNonFirstAlphabet(c) {
		scan.step = fs.stateInComplexPath
		return scanFilterContinue
	}

	return fs.error(c, "invalid character in complex path")
}

// Intermediate state where we are inside an attribute path name with operators ex. emails[type eq "work"],  an ] character would end the path name and start
// an operator.
func (fs *filterScanner) stateInComplexPath(scan *filterScanner, c byte) int {
	if c == ' ' {
		scan.step = fs.stateBeginComplexOp
		return scanFilterContinue
	}

	if c == '.' || c == ':' || isNonFirstAlphabet(c) {
		return scanFilterContinue
	}

	return fs.error(c, "invalid character in complex path")
}

// Intermediate state at the end of  an attribute path name with operators ex. emails[type eq "work"],  an ] character would end the path name and start
// an operator or end the scan if this was the only path entered to the filter.
func (fs *filterScanner) stateEndComplexPath(scan *filterScanner, c byte) int {
	if c == ' ' {
		scan.step = fs.stateBeginComplexOp
		return scanFilterEndPath
	}

	if c == 0 {
		scan.step = fs.stateEndComplexPredicate
		return scanFilterEnd
	}

	return fs.error(c, "invalid character in complex path")
}

// Intermediate state at the beginning of an operator defined by SCIM query protocol.
func (fs *filterScanner) stateBeginComplexOp(scan *filterScanner, c byte) int {
	if c == ' ' {
		return scanFilterSkipSpace
	}

	switch c {
	case 'a', 'A':
		// and
		scan.step = fs.stateComplexOpA
		return scanFilterContinue
	case 'c', 'C':
		// co
		scan.step = fs.stateComplexOpC
		return scanFilterContinue
	case 'e', 'E':
		// eq, ew
		scan.step = fs.stateComplexOpE
		return scanFilterContinue
	case 'g', 'G':
		// gt, ge
		scan.step = fs.stateComplexOpG
		return scanFilterContinue
	case 'l', 'L':
		// lt, le
		scan.step = fs.stateComplexOpL
		return scanFilterContinue
	case 'n', 'N':
		// not, ne
		scan.step = fs.stateComplexOpN
		return scanFilterContinue
	case 'o', 'O':
		// or
		scan.step = fs.stateComplexOpO
		return scanFilterContinue
	case 'p', 'P':
		// pr
		scan.step = fs.stateComplexOpP
		return scanFilterContinue
	case 's', 'S':
		// sw
		scan.step = fs.stateComplexOpS
		return scanFilterContinue
	}

	return fs.error(c, "invalid character in operator")
}

// Intermediate state in operator where the last character was 'a' (case insensitive). The current character must be
// 'n' (case insensitive) to lead to the logical and operator.
func (fs *filterScanner) stateComplexOpA(scan *filterScanner, c byte) int {
	if c == 'n' || c == 'N' {
		scan.step = fs.stateComplexOpAn
		return scanFilterContinue
	}
	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where the last two characters were 'a' and 'n' (case insensitive). The current
// character must be 'd' (case insensitive) to lead to the logical and operator.
func (fs *filterScanner) stateComplexOpAn(scan *filterScanner, c byte) int {
	if c == 'd' || c == 'D' {
		scan.step = fs.stateComplexOpAnd
		return scanFilterContinue
	}
	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where the last three characters were 'a', 'n' and 'd' (case insensitive). The current
// character must end the operator.
func (fs *filterScanner) stateComplexOpAnd(scan *filterScanner, c byte) int {
	if c == ' ' {
		scan.step = fs.stateBeginComplexPredicate
		return scanFilterContinue
	}

	if c == '(' {
		return scanFilterInsertSpace
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where the last character was 'c' (case insensitive). The current character must be
// 'o' (case insensitive) to lead to a relational co operator.
func (fs *filterScanner) stateComplexOpC(scan *filterScanner, c byte) int {
	if c == 'o' || c == 'O' {
		scan.step = fs.stateComplexOpCo
		return scanFilterContinue
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where the last two characters were 'c' and 'o' (case insensitive). The current
// character must be space to end the operator.
func (fs *filterScanner) stateComplexOpCo(scan *filterScanner, c byte) int {
	if c == ' ' {
		scan.step = fs.stateBeginComplexLiteral
		return scanFilterContinue
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where last character was 'e' (case insensitive). The current character should be
// 'q' or 'w' (case insensitive) to lead to eq/ew relational operator.
func (fs *filterScanner) stateComplexOpE(scan *filterScanner, c byte) int {
	switch c {
	case 'q', 'Q':
		scan.step = fs.stateComplexOpEq
		return scanFilterContinue
	case 'w', 'W':
		scan.step = fs.stateComplexOpEw
		return scanFilterContinue
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where last two characters were 'e' and 'q' (case insensitive). The current character
// must end the operator with space.
func (fs *filterScanner) stateComplexOpEq(scan *filterScanner, c byte) int {
	if c == ' ' {
		scan.step = fs.stateBeginComplexLiteral
		return scanFilterContinue
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where last two characters were 'e' and 'w' (case insensitive). The current character
// must end the operator with space.
func (fs *filterScanner) stateComplexOpEw(scan *filterScanner, c byte) int {
	if c == ' ' {
		scan.step = fs.stateBeginComplexLiteral
		return scanFilterContinue
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where last character was 'g' (case insensitive). The current character should be
// 't' or 'e' (case insensitive) to lead to gt/ge relational operator.
func (fs *filterScanner) stateComplexOpG(scan *filterScanner, c byte) int {
	switch c {
	case 't', 'T':
		scan.step = fs.stateComplexOpGt
		return scanFilterContinue
	case 'e', 'E':
		scan.step = fs.stateComplexOpGe
		return scanFilterContinue
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where last two characters were 'g' and 't' (case insensitive). The current character
// must end the operator with space.
func (fs *filterScanner) stateComplexOpGt(scan *filterScanner, c byte) int {
	if c == ' ' {
		scan.step = fs.stateBeginComplexLiteral
		return scanFilterContinue
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where last two characters were 'g' and 'e' (case insensitive). The current character
// must end the operator with space.
func (fs *filterScanner) stateComplexOpGe(scan *filterScanner, c byte) int {
	if c == ' ' {
		scan.step = fs.stateBeginComplexLiteral
		return scanFilterContinue
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where last character was 'l' (case insensitive). The current character should be
// 't' or 'e' (case insensitive) to lead to gt/ge relational operator.
func (fs *filterScanner) stateComplexOpL(scan *filterScanner, c byte) int {
	switch c {
	case 't', 'T':
		scan.step = fs.stateComplexOpLt
		return scanFilterContinue
	case 'e', 'E':
		scan.step = fs.stateComplexOpLe
		return scanFilterContinue
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where last two characters were 'l' and 't' (case insensitive). The current character
// must end the operator with space.
func (fs *filterScanner) stateComplexOpLt(scan *filterScanner, c byte) int {
	if c == ' ' {
		scan.step = fs.stateBeginComplexLiteral
		return scanFilterContinue
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where last two characters were 'l' and 'e' (case insensitive). The current character
// must end the operator with space.
func (fs *filterScanner) stateComplexOpLe(scan *filterScanner, c byte) int {
	if c == ' ' {
		scan.step = fs.stateBeginComplexLiteral
		return scanFilterContinue
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where last character was 'n' (case insensitive). The current character should be
// 'o' or 'e' (case insensitive) to lead to 'not' logical operator or 'ne' relational operator.
func (fs *filterScanner) stateComplexOpN(scan *filterScanner, c byte) int {
	switch c {
	case 'o', 'O':
		scan.step = fs.stateComplexOpNo
		return scanFilterContinue
	case 'e', 'E':
		scan.step = fs.stateComplexOpNe
		return scanFilterContinue
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where last two characters were 'n' and 'e' (case insensitive). The current character
// must end the operator with space.
func (fs *filterScanner) stateComplexOpNe(scan *filterScanner, c byte) int {
	if c == ' ' {
		scan.step = fs.stateBeginComplexLiteral
		return scanFilterContinue
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where last two characters were 'n' and 'o' (case insensitive). The current character
// must be 't' (case insensitive) to lead to 'not' logical operator.
func (fs *filterScanner) stateComplexOpNo(scan *filterScanner, c byte) int {
	if c == 't' || c == 'T' {
		scan.step = fs.stateComplexOpNot
		return scanFilterContinue
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where last three characters were 'n', 'o' and 't' (case insensitive). The current
// character must end the operator.
func (fs *filterScanner) stateComplexOpNot(scan *filterScanner, c byte) int {
	if c == ' ' {
		scan.step = fs.stateBeginComplexPredicate
		return scanFilterContinue
	}

	if c == '(' {
		return scanFilterInsertSpace
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where last character was 'o' (case insensitive). The current character should be
// 'r' (case insensitive) to lead to 'or' logical operator.
func (fs *filterScanner) stateComplexOpO(scan *filterScanner, c byte) int {
	if c == 'r' || c == 'R' {
		scan.step = fs.stateComplexOpOr
		return scanFilterContinue
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where last two characters were 'o' and 'r' (case insensitive). The current character
// must end the operator.
func (fs *filterScanner) stateComplexOpOr(scan *filterScanner, c byte) int {
	if c == ' ' {
		scan.step = fs.stateBeginComplexPredicate
		return scanFilterContinue
	}

	if c == '(' {
		return scanFilterInsertSpace
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where last character was 'p' (case insensitive). The current character should be
// 'r' (case insensitive) to lead to 'or' logical operator.
func (fs *filterScanner) stateComplexOpP(scan *filterScanner, c byte) int {
	if c == 'r' || c == 'R' {
		scan.step = fs.stateComplexOpPr
		return scanFilterContinue
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where last two characters were 'p' and 'r' (case insensitive). The current character
// must end the predicate.
func (fs *filterScanner) stateComplexOpPr(scan *filterScanner, c byte) int {
	if c == ' ' || c == 0 {
		scan.step = fs.stateEndComplexPredicate
		return scanFilterContinue
	}

	if c == ')' {
		return scanFilterInsertSpace
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where last character was 'e' (case insensitive). The current character should be
// 'w' (case insensitive) to lead to sw relational operator.
func (fs *filterScanner) stateComplexOpS(scan *filterScanner, c byte) int {
	if c == 'w' || c == 'W' {
		scan.step = fs.stateComplexOpSw
		return scanFilterContinue
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state in operator where last two characters were 's' and 'w' (case insensitive). The current character
// must end the operator with space.
func (fs *filterScanner) stateComplexOpSw(scan *filterScanner, c byte) int {
	if c == ' ' {
		scan.step = fs.stateBeginComplexLiteral
		return scanFilterContinue
	}

	return fs.errInvalidOperator(c)
}

// Intermediate state at the start of a literal. We distinguish between string and non-string literal.
func (fs *filterScanner) stateBeginComplexLiteral(scan *filterScanner, c byte) int {
	switch c {
	case '"':
		scan.step = fs.stateInComplexStringLiteral
		return scanFilterContinue
	case 't', 'T', 'f', 'F', '-', '+', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		scan.step = fs.stateInComplexNonStringLiteral
		return scanFilterContinue
	}

	return fs.error(c, "invalid literal")
}

// Intermediate state at the end of a literal.
func (fs *filterScanner) stateEndComplexLiteral(scan *filterScanner, c byte) int {
	if c == ' ' {
		return scanFilterSkipSpace
	}

	if c == ')' {
		fs.parenLevel--
		scan.step = fs.stateEndComplexPredicate
		return scanFilterContinue
	}

	if c == 0 {
		scan.step = fs.stateEOF
		return scanFilterEnd
	}

	// ending a literal can also mean ending a predicate implicitly
	return fs.stateEndComplexPredicate(scan, c)
}

// Intermediate state in a string literal.
func (fs *filterScanner) stateInComplexStringLiteral(scan *filterScanner, c byte) int {
	if c == '\\' {
		scan.step = fs.stateInComplexStringEsc
	}

	if c == '"' {
		scan.step = fs.stateEndComplexStringLiteral
	}

	return scanFilterContinue
}

// Intermediate state after ending a string literal with double quote. This state is necessary so that the reported
// events can be easily interpreted against the index to produce a string literal with starting and ending double quotes.
func (fs *filterScanner) stateEndComplexStringLiteral(scan *filterScanner, c byte) int {
	switch c {
	case ' ':
		scan.step = fs.stateEndComplexLiteral
		return scanFilterContinue
	case ')':
		return scanFilterInsertSpace
	case ']':
		scan.step = fs.stateEndComplexPath
		return scanFilterContinue
	case 0:
		scan.step = fs.stateEOF
		return scanFilterEndLiteral
	}

	return fs.error(c, "invalid character trailing string literal")
}

// Intermediate state in a non-string literal. Here, we only care about termination of the literal.
func (fs *filterScanner) stateInComplexNonStringLiteral(scan *filterScanner, c byte) int {
	switch c {
	case ' ':
		scan.step = fs.stateEndComplexLiteral
		return scanFilterEndLiteral
	case ')':
		return scanFilterInsertSpace
	case 0:
		scan.step = fs.stateEOF
		return scanFilterEndLiteral
	default:
		return scanFilterContinue
	}
}

// Intermediate state where we are inside an escaped string. Regular escape character return the state to stateInString.
// A unicode escape character (i.e \u0000) enter the state into escaped unicode string.
func (fs *filterScanner) stateInComplexStringEsc(_ *filterScanner, c byte) int {
	switch c {
	case 'b', 'f', 'n', 'r', 't', '\\', '/', '"':
		fs.step = fs.stateInComplexStringLiteral
		return scanFilterContinue
	case 'u':
		fs.step = fs.stateInComplexStringEscU
		return scanFilterContinue
	}
	return fs.error(c, "invalid character in string literal")
}

// Intermediate state where we are at the leading byte of the 4 byte unicode.
func (fs *filterScanner) stateInComplexStringEscU(_ *filterScanner, c byte) int {
	if '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F' {
		fs.step = fs.stateInComplexStringEscU1
		return scanFilterContinue
	}

	return fs.error(c, "in \\u hexadecimal character escape")
}

// Intermediate state where we are at the second leading byte of the 4 byte unicode.
func (fs *filterScanner) stateInComplexStringEscU1(_ *filterScanner, c byte) int {
	if '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F' {
		fs.step = fs.stateInComplexStringEscU12
		return scanFilterContinue
	}

	return fs.error(c, "in \\u hexadecimal character escape")
}

// Intermediate state where we are at the third leading byte of the 4 byte unicode.
func (fs *filterScanner) stateInComplexStringEscU12(_ *filterScanner, c byte) int {
	if '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F' {
		fs.step = fs.stateInComplexStringEscU123
		return scanFilterContinue
	}

	return fs.error(c, "in \\u hexadecimal character escape")
}

// Intermediate state where we are at the last byte of the 4 byte unicode.
func (fs *filterScanner) stateInComplexStringEscU123(_ *filterScanner, c byte) int {
	if '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F' {
		fs.step = fs.stateInComplexStringLiteral
		return scanFilterContinue
	}

	return fs.error(c, "in \\u hexadecimal character escape")
}
