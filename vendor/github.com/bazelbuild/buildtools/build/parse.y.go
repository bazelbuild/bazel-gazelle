//line build/parse.y:13
package build

import __yyfmt__ "fmt"

//line build/parse.y:13

//line build/parse.y:18
type yySymType struct {
	yys int
	// input tokens
	tok    string   // raw input syntax
	str    string   // decoding of quoted string
	pos    Position // position of token
	triple bool     // was string triple quoted?

	// partial syntax trees
	expr    Expr
	exprs   []Expr
	kv      *KeyValueExpr
	kvs     []*KeyValueExpr
	string  *StringExpr
	ifstmt  *IfStmt
	loadarg *struct {
		from Ident
		to   Ident
	}
	loadargs []*struct {
		from Ident
		to   Ident
	}

	// supporting information
	comma    Position // position of trailing comma in list, if present
	lastStmt Expr     // most recent rule, to attach line comments to
}

const _AUGM = 57346
const _AND = 57347
const _COMMENT = 57348
const _EOF = 57349
const _EQ = 57350
const _FOR = 57351
const _GE = 57352
const _IDENT = 57353
const _INT = 57354
const _IF = 57355
const _ELSE = 57356
const _ELIF = 57357
const _IN = 57358
const _IS = 57359
const _LAMBDA = 57360
const _LOAD = 57361
const _LE = 57362
const _NE = 57363
const _STAR_STAR = 57364
const _INT_DIV = 57365
const _BIT_LSH = 57366
const _BIT_RSH = 57367
const _NOT = 57368
const _OR = 57369
const _STRING = 57370
const _DEF = 57371
const _RETURN = 57372
const _PASS = 57373
const _BREAK = 57374
const _CONTINUE = 57375
const _INDENT = 57376
const _UNINDENT = 57377
const ShiftInstead = 57378
const _ASSERT = 57379
const _UNARY = 57380

var yyToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"'%'",
	"'('",
	"')'",
	"'*'",
	"'+'",
	"','",
	"'-'",
	"'.'",
	"'/'",
	"':'",
	"'<'",
	"'='",
	"'>'",
	"'['",
	"']'",
	"'{'",
	"'}'",
	"'|'",
	"'&'",
	"'^'",
	"'~'",
	"_AUGM",
	"_AND",
	"_COMMENT",
	"_EOF",
	"_EQ",
	"_FOR",
	"_GE",
	"_IDENT",
	"_INT",
	"_IF",
	"_ELSE",
	"_ELIF",
	"_IN",
	"_IS",
	"_LAMBDA",
	"_LOAD",
	"_LE",
	"_NE",
	"_STAR_STAR",
	"_INT_DIV",
	"_BIT_LSH",
	"_BIT_RSH",
	"_NOT",
	"_OR",
	"_STRING",
	"_DEF",
	"_RETURN",
	"_PASS",
	"_BREAK",
	"_CONTINUE",
	"_INDENT",
	"_UNINDENT",
	"ShiftInstead",
	"'\\n'",
	"_ASSERT",
	"_UNARY",
	"';'",
}
var yyStatenames = [...]string{}

const yyEofCode = 1
const yyErrCode = 2
const yyInitialStackSize = 16

//line build/parse.y:957

// Go helper code.

// unary returns a unary expression with the given
// position, operator, and subexpression.
func unary(pos Position, op string, x Expr) Expr {
	return &UnaryExpr{
		OpStart: pos,
		Op:      op,
		X:       x,
	}
}

// binary returns a binary expression with the given
// operands, position, and operator.
func binary(x Expr, pos Position, op string, y Expr) Expr {
	_, xend := x.Span()
	ystart, _ := y.Span()

	switch op {
	case "=", "+=", "-=", "*=", "/=", "//=", "%=", "|=":
		return &AssignExpr{
			LHS:       x,
			OpPos:     pos,
			Op:        op,
			LineBreak: xend.Line < ystart.Line,
			RHS:       y,
		}
	}

	return &BinaryExpr{
		X:         x,
		OpStart:   pos,
		Op:        op,
		LineBreak: xend.Line < ystart.Line,
		Y:         y,
	}
}

// isSimpleExpression returns whether an expression is simple and allowed to exist in
// compact forms of sequences.
// The formal criteria are the following: an expression is considered simple if it's
// a literal (variable, string or a number), a literal with a unary operator or an empty sequence.
func isSimpleExpression(expr *Expr) bool {
	switch x := (*expr).(type) {
	case *LiteralExpr, *StringExpr, *Ident:
		return true
	case *UnaryExpr:
		_, literal := x.X.(*LiteralExpr)
		_, ident := x.X.(*Ident)
		return literal || ident
	case *ListExpr:
		return len(x.List) == 0
	case *TupleExpr:
		return len(x.List) == 0
	case *DictExpr:
		return len(x.List) == 0
	case *SetExpr:
		return len(x.List) == 0
	default:
		return false
	}
}

// forceCompact returns the setting for the ForceCompact field for a call or tuple.
//
// NOTE 1: The field is called ForceCompact, not ForceSingleLine,
// because it only affects the formatting associated with the call or tuple syntax,
// not the formatting of the arguments. For example:
//
//	call([
//		1,
//		2,
//		3,
//	])
//
// is still a compact call even though it runs on multiple lines.
//
// In contrast the multiline form puts a linebreak after the (.
//
//	call(
//		[
//			1,
//			2,
//			3,
//		],
//	)
//
// NOTE 2: Because of NOTE 1, we cannot use start and end on the
// same line as a signal for compact mode: the formatting of an
// embedded list might move the end to a different line, which would
// then look different on rereading and cause buildifier not to be
// idempotent. Instead, we have to look at properties guaranteed
// to be preserved by the reformatting, namely that the opening
// paren and the first expression are on the same line and that
// each subsequent expression begins on the same line as the last
// one ended (no line breaks after comma).
func forceCompact(start Position, list []Expr, end Position) bool {
	if len(list) <= 1 {
		// The call or tuple will probably be compact anyway; don't force it.
		return false
	}

	// If there are any named arguments or non-string, non-literal
	// arguments, cannot force compact mode.
	line := start.Line
	for _, x := range list {
		start, end := x.Span()
		if start.Line != line {
			return false
		}
		line = end.Line
		if !isSimpleExpression(&x) {
			return false
		}
	}
	return end.Line == line
}

// forceMultiLine returns the setting for the ForceMultiLine field.
func forceMultiLine(start Position, list []Expr, end Position) bool {
	if len(list) > 1 {
		// The call will be multiline anyway, because it has multiple elements. Don't force it.
		return false
	}

	if len(list) == 0 {
		// Empty list: use position of brackets.
		return start.Line != end.Line
	}

	// Single-element list.
	// Check whether opening bracket is on different line than beginning of
	// element, or closing bracket is on different line than end of element.
	elemStart, elemEnd := list[0].Span()
	return start.Line != elemStart.Line || end.Line != elemEnd.Line
}

// forceMultiLineComprehension returns the setting for the ForceMultiLine field for a comprehension.
func forceMultiLineComprehension(start Position, expr Expr, clauses []Expr, end Position) bool {
	// Return true if there's at least one line break between start, expr, each clause, and end
	exprStart, exprEnd := expr.Span()
	if start.Line != exprStart.Line {
		return true
	}
	previousEnd := exprEnd
	for _, clause := range clauses {
		clauseStart, clauseEnd := clause.Span()
		if previousEnd.Line != clauseStart.Line {
			return true
		}
		previousEnd = clauseEnd
	}
	return previousEnd.Line != end.Line
}

// extractTrailingComments extracts trailing comments of an indented block starting with the first
// comment line with indentation less than the block indentation.
// The comments can either belong to CommentBlock statements or to the last non-comment statement
// as After-comments.
func extractTrailingComments(stmt Expr) []Expr {
	body := getLastBody(stmt)
	var comments []Expr
	if body != nil && len(*body) > 0 {
		// Get the current indentation level
		start, _ := (*body)[0].Span()
		indentation := start.LineRune

		// Find the last non-comment statement
		lastNonCommentIndex := -1
		for i, stmt := range *body {
			if _, ok := stmt.(*CommentBlock); !ok {
				lastNonCommentIndex = i
			}
		}
		if lastNonCommentIndex == -1 {
			return comments
		}

		// Iterate over the trailing comments, find the first comment line that's not indented enough,
		// dedent it and all the following comments.
		for i := lastNonCommentIndex; i < len(*body); i++ {
			stmt := (*body)[i]
			if comment := extractDedentedComment(stmt, indentation); comment != nil {
				// This comment and all the following CommentBlock statements are to be extracted.
				comments = append(comments, comment)
				comments = append(comments, (*body)[i+1:]...)
				*body = (*body)[:i+1]
				// If the current statement is a CommentBlock statement without any comment lines
				// it should be removed too.
				if i > lastNonCommentIndex && len(stmt.Comment().After) == 0 {
					*body = (*body)[:i]
				}
			}
		}
	}
	return comments
}

// extractDedentedComment extract the first comment line from `stmt` which indentation is smaller
// than `indentation`, and all following comment lines, and returns them in a newly created
// CommentBlock statement.
func extractDedentedComment(stmt Expr, indentation int) Expr {
	for i, line := range stmt.Comment().After {
		// line.Start.LineRune == 0 can't exist in parsed files, it indicates that the comment line
		// has been added by an AST modification. Don't take such lines into account.
		if line.Start.LineRune > 0 && line.Start.LineRune < indentation {
			// This and all the following lines should be dedented
			cb := &CommentBlock{
				Start:    line.Start,
				Comments: Comments{After: stmt.Comment().After[i:]},
			}
			stmt.Comment().After = stmt.Comment().After[:i]
			return cb
		}
	}
	return nil
}

// getLastBody returns the last body of a block statement (the only body for For- and DefStmt
// objects, the last in a if-elif-else chain
func getLastBody(stmt Expr) *[]Expr {
	switch block := stmt.(type) {
	case *DefStmt:
		return &block.Body
	case *ForStmt:
		return &block.Body
	case *IfStmt:
		if len(block.False) == 0 {
			return &block.True
		} else if len(block.False) == 1 {
			if next, ok := block.False[0].(*IfStmt); ok {
				// Recursively find the last block of the chain
				return getLastBody(next)
			}
		}
		return &block.False
	}
	return nil
}

//line yacctab:1
var yyExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const yyPrivate = 57344

const yyLast = 874

var yyAct = [...]int{

	19, 174, 29, 208, 7, 91, 27, 211, 146, 138,
	21, 154, 79, 172, 77, 2, 41, 9, 153, 101,
	230, 221, 42, 166, 83, 84, 85, 86, 38, 80,
	37, 34, 89, 94, 97, 124, 104, 51, 48, 49,
	50, 54, 192, 55, 165, 52, 107, 219, 37, 81,
	88, 111, 112, 113, 114, 115, 116, 117, 118, 119,
	120, 121, 122, 123, 215, 125, 126, 127, 128, 129,
	130, 131, 132, 133, 34, 218, 139, 53, 220, 13,
	104, 99, 140, 109, 96, 82, 34, 135, 149, 150,
	151, 40, 156, 90, 46, 51, 196, 195, 50, 158,
	73, 190, 161, 52, 110, 45, 33, 156, 103, 80,
	159, 43, 36, 156, 160, 170, 75, 168, 31, 167,
	32, 44, 74, 105, 106, 175, 93, 108, 76, 45,
	236, 223, 152, 34, 35, 53, 222, 226, 184, 181,
	148, 30, 182, 183, 216, 185, 179, 45, 177, 188,
	37, 201, 145, 45, 80, 45, 143, 171, 194, 134,
	187, 98, 225, 197, 199, 191, 189, 42, 180, 198,
	45, 191, 51, 193, 163, 50, 54, 205, 55, 157,
	52, 139, 207, 169, 147, 202, 209, 140, 232, 200,
	206, 178, 212, 214, 164, 102, 213, 87, 100, 204,
	1, 210, 203, 51, 95, 92, 50, 54, 39, 55,
	47, 52, 53, 69, 70, 217, 227, 18, 12, 224,
	66, 67, 68, 8, 209, 186, 4, 231, 212, 234,
	228, 7, 213, 233, 229, 33, 28, 155, 25, 144,
	24, 36, 78, 53, 69, 70, 136, 31, 137, 32,
	0, 0, 0, 0, 26, 0, 0, 6, 0, 0,
	11, 0, 34, 35, 20, 0, 0, 0, 0, 22,
	30, 0, 0, 0, 0, 0, 0, 23, 0, 37,
	10, 14, 15, 16, 17, 0, 235, 33, 5, 0,
	25, 0, 24, 36, 0, 0, 0, 0, 0, 31,
	0, 32, 0, 0, 0, 0, 26, 0, 0, 6,
	3, 0, 11, 0, 34, 35, 20, 0, 0, 0,
	0, 22, 30, 0, 0, 0, 0, 0, 0, 23,
	0, 37, 10, 14, 15, 16, 17, 0, 33, 0,
	5, 25, 0, 24, 36, 0, 0, 0, 0, 0,
	31, 0, 32, 0, 0, 0, 0, 26, 0, 0,
	0, 0, 0, 0, 0, 34, 35, 0, 0, 0,
	0, 0, 22, 30, 0, 0, 0, 0, 0, 0,
	23, 0, 37, 0, 14, 15, 16, 17, 0, 51,
	0, 173, 50, 54, 0, 55, 0, 52, 162, 56,
	0, 57, 0, 0, 0, 0, 66, 67, 68, 0,
	0, 65, 0, 0, 58, 0, 61, 0, 0, 72,
	0, 0, 62, 71, 0, 0, 59, 60, 0, 53,
	69, 70, 63, 64, 51, 0, 0, 50, 54, 0,
	55, 0, 52, 0, 56, 0, 57, 0, 0, 0,
	0, 66, 67, 68, 0, 0, 65, 0, 0, 58,
	0, 61, 0, 0, 72, 176, 0, 62, 71, 0,
	0, 59, 60, 0, 53, 69, 70, 63, 64, 51,
	0, 0, 50, 54, 0, 55, 0, 52, 0, 56,
	0, 57, 0, 0, 0, 0, 66, 67, 68, 0,
	0, 65, 0, 0, 58, 156, 61, 0, 0, 72,
	0, 0, 62, 71, 0, 0, 59, 60, 0, 53,
	69, 70, 63, 64, 51, 0, 0, 50, 54, 0,
	55, 0, 52, 0, 56, 0, 57, 0, 0, 0,
	0, 66, 67, 68, 0, 0, 65, 0, 0, 58,
	0, 61, 0, 0, 72, 0, 0, 62, 71, 0,
	0, 59, 60, 0, 53, 69, 70, 63, 64, 33,
	0, 0, 25, 0, 24, 36, 0, 0, 0, 0,
	0, 31, 0, 32, 0, 0, 0, 0, 26, 0,
	0, 0, 0, 0, 0, 0, 34, 35, 0, 0,
	0, 0, 51, 22, 30, 50, 54, 0, 55, 0,
	52, 23, 56, 37, 57, 14, 15, 16, 17, 66,
	67, 68, 0, 0, 65, 0, 0, 58, 0, 61,
	0, 0, 0, 0, 0, 62, 71, 0, 0, 59,
	60, 0, 53, 69, 70, 63, 64, 51, 0, 0,
	50, 54, 0, 55, 0, 52, 0, 56, 0, 57,
	0, 0, 0, 0, 66, 67, 68, 0, 0, 65,
	0, 0, 58, 0, 61, 0, 0, 0, 0, 0,
	62, 0, 0, 0, 59, 60, 0, 53, 69, 70,
	63, 64, 51, 0, 0, 50, 54, 0, 55, 0,
	52, 0, 56, 0, 57, 0, 0, 0, 0, 66,
	67, 68, 0, 0, 65, 0, 0, 58, 0, 61,
	0, 0, 0, 0, 0, 62, 0, 0, 0, 59,
	60, 0, 53, 69, 70, 63, 51, 0, 0, 50,
	54, 0, 55, 0, 52, 0, 56, 0, 57, 0,
	0, 0, 0, 66, 67, 68, 0, 0, 0, 0,
	0, 58, 0, 61, 0, 0, 0, 0, 0, 62,
	0, 0, 0, 59, 60, 0, 53, 69, 70, 63,
	33, 0, 141, 25, 0, 24, 36, 51, 0, 0,
	50, 54, 31, 55, 32, 52, 0, 0, 33, 26,
	0, 25, 0, 24, 36, 67, 68, 34, 35, 0,
	31, 0, 32, 0, 22, 30, 0, 26, 142, 0,
	0, 0, 23, 0, 37, 34, 35, 53, 69, 70,
	0, 51, 22, 30, 50, 54, 0, 55, 0, 52,
	23, 0, 37, 0, 0, 0, 0, 0, 0, 67,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 53, 69, 70,
}
var yyPact = [...]int{

	-1000, -1000, 282, -1000, -1000, -1000, -30, -1000, -1000, -1000,
	59, 101, -1000, 96, 793, -1000, -1000, -1000, 3, 520,
	793, 111, 42, 793, 793, 793, 793, -1000, -1000, -1000,
	192, 793, 793, 793, -1000, 150, 48, -1000, -1000, -42,
	190, 71, 111, 793, 793, 793, 161, 793, 70, -1000,
	793, 793, 793, 793, 793, 793, 793, 793, 793, 793,
	793, 793, 793, -2, 793, 793, 793, 793, 793, 793,
	793, 793, 793, 146, 55, 775, 793, 139, 175, -1000,
	125, 54, 54, -1000, -1000, -1000, -1000, -19, 114, 475,
	170, 62, 94, 170, 385, 165, 188, 520, 11, -1000,
	-35, 564, 42, 793, 101, 161, 161, 598, 144, 333,
	-1000, -1000, -1000, -1000, -1000, 91, 91, 199, 199, 199,
	199, 199, 199, 199, 793, 688, 732, 783, 168, 827,
	33, 33, 643, 430, 333, -1000, 185, 159, -1000, 520,
	124, 793, 793, 120, 132, 793, -1000, 42, 793, -1000,
	-1000, 157, -1000, 83, 8, -1000, 101, 793, -1000, 77,
	-1000, 76, 793, 793, -1000, -1000, -1000, -1000, 183, 138,
	111, 333, -1000, -1000, -1000, 199, 793, -1000, -1000, -1000,
	775, 793, 520, 520, -1000, 793, -1000, -1000, 520, -1,
	-1000, 8, 793, 27, 520, -1000, -1000, 520, -1000, 385,
	131, 333, -1000, 20, -37, 598, -1000, 520, 118, 520,
	153, -1000, -1000, 122, 598, 793, 333, -1000, -1000, -38,
	-1000, -1000, -1000, 793, 182, -1, -19, 598, -1000, 230,
	-1000, 112, -1000, -1000, -1000, -1000, -1000,
}
var yyPgo = [...]int{

	0, 8, 9, 248, 246, 12, 242, 14, 0, 3,
	50, 10, 79, 239, 93, 16, 237, 11, 18, 6,
	236, 15, 226, 223, 218, 217, 210, 1, 17, 208,
	5, 205, 204, 2, 13, 202, 7, 201, 200, 199,
	198,
}
var yyR1 = [...]int{

	0, 38, 34, 34, 39, 39, 35, 35, 35, 21,
	21, 21, 21, 22, 22, 23, 23, 23, 25, 25,
	24, 24, 26, 26, 27, 29, 29, 28, 28, 28,
	28, 28, 28, 28, 28, 40, 40, 11, 11, 11,
	11, 11, 11, 11, 11, 11, 11, 11, 11, 11,
	11, 11, 4, 4, 3, 3, 2, 2, 2, 2,
	37, 37, 36, 36, 7, 7, 6, 6, 5, 5,
	5, 5, 5, 12, 12, 13, 13, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 14, 14, 9, 9,
	10, 10, 1, 1, 30, 32, 32, 31, 31, 31,
	15, 15, 33, 19, 20, 20, 20, 20, 16, 17,
	17, 18, 18,
}
var yyR2 = [...]int{

	0, 2, 5, 2, 0, 2, 0, 3, 2, 0,
	2, 2, 3, 1, 1, 7, 6, 1, 4, 5,
	1, 4, 2, 1, 4, 0, 3, 1, 2, 1,
	3, 3, 1, 1, 1, 0, 1, 1, 1, 1,
	3, 7, 4, 4, 6, 8, 3, 4, 4, 3,
	4, 3, 0, 2, 1, 3, 1, 3, 2, 2,
	1, 3, 1, 3, 0, 2, 1, 3, 1, 3,
	2, 1, 2, 1, 3, 0, 1, 1, 4, 2,
	2, 2, 2, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 4, 3, 3, 3,
	3, 3, 3, 3, 3, 5, 1, 3, 0, 1,
	0, 2, 0, 1, 3, 1, 3, 0, 1, 2,
	1, 3, 1, 1, 3, 2, 2, 1, 4, 1,
	3, 1, 2,
}
var yyChk = [...]int{

	-1000, -38, -21, 28, -22, 58, 27, -27, -23, -28,
	50, 30, -24, -12, 51, 52, 53, 54, -25, -8,
	34, -11, 39, 47, 10, 8, 24, -19, -20, -33,
	40, 17, 19, 5, 32, 33, 11, 49, 58, -29,
	32, -15, -11, 15, 25, 9, -12, -26, 35, 36,
	7, 4, 12, 44, 8, 10, 14, 16, 29, 41,
	42, 31, 37, 47, 48, 26, 21, 22, 23, 45,
	46, 38, 34, -12, 11, 5, 17, -7, -6, -5,
	-19, 7, 43, -8, -8, -8, -8, 5, -10, -8,
	-14, -30, -31, -14, -8, -32, -10, -8, 11, 33,
	-40, 61, 5, 37, 9, -12, -12, -8, -12, 13,
	34, -8, -8, -8, -8, -8, -8, -8, -8, -8,
	-8, -8, -8, -8, 37, -8, -8, -8, -8, -8,
	-8, -8, -8, -8, 13, 32, -4, -3, -2, -8,
	-19, 7, 43, -12, -13, 13, -1, 9, 15, -19,
	-19, -33, 18, -18, -17, -16, 30, 9, -1, -18,
	20, -1, 13, 9, 6, 33, 58, -28, -7, -12,
	-11, 13, -34, 58, -27, -8, 35, -34, 6, -1,
	9, 15, -8, -8, 18, 13, -12, -5, -8, 9,
	18, -17, 34, -15, -8, 20, 20, -8, -30, -8,
	6, 13, -34, -35, -39, -8, -2, -8, -9, -8,
	-37, -36, -33, -19, -8, 37, 13, -34, 55, 27,
	58, 58, 18, 13, -1, 9, 15, -8, -34, -21,
	58, -9, 6, -36, -33, 56, 18,
}
var yyDef = [...]int{

	9, -2, 0, 1, 10, 11, 0, 13, 14, 25,
	0, 0, 17, 27, 29, 32, 33, 34, 20, 73,
	0, 77, 64, 0, 0, 0, 0, 37, 38, 39,
	0, 110, 117, 110, 123, 127, 0, 122, 12, 35,
	0, 0, 120, 0, 0, 0, 28, 0, 0, 23,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 52, 75, 0, 112, 66,
	68, 71, 0, 79, 80, 81, 82, 0, 0, 106,
	112, 115, 0, 112, 106, 118, 0, 106, 125, 126,
	0, 36, 64, 0, 0, 30, 31, 74, 0, 0,
	22, 83, 84, 85, 86, 87, 88, 89, 90, 91,
	92, 93, 94, 95, 0, 97, 98, 99, 100, 101,
	102, 103, 104, 0, 0, 40, 0, 112, 54, 56,
	37, 0, 0, 76, 0, 0, 65, 113, 0, 70,
	72, 0, 46, 0, 131, 129, 0, 113, 111, 0,
	49, 0, 0, 119, 51, 124, 24, 26, 0, 0,
	121, 0, 21, 6, 4, 96, 0, 18, 42, 53,
	113, 0, 58, 59, 43, 108, 78, 67, 69, 0,
	47, 132, 0, 0, 107, 48, 50, 114, 116, 0,
	0, 0, 19, 0, 3, 105, 55, 57, 0, 109,
	112, 60, 62, 0, 130, 0, 0, 16, 9, 0,
	8, 5, 44, 108, 0, 113, 0, 128, 15, 0,
	7, 0, 41, 61, 63, 2, 45,
}
var yyTok1 = [...]int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	58, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 4, 22, 3,
	5, 6, 7, 8, 9, 10, 11, 12, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 13, 61,
	14, 15, 16, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 17, 3, 18, 23, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 19, 21, 20, 24,
}
var yyTok2 = [...]int{

	2, 3, 25, 26, 27, 28, 29, 30, 31, 32,
	33, 34, 35, 36, 37, 38, 39, 40, 41, 42,
	43, 44, 45, 46, 47, 48, 49, 50, 51, 52,
	53, 54, 55, 56, 57, 59, 60,
}
var yyTok3 = [...]int{
	0,
}

var yyErrorMessages = [...]struct {
	state int
	token int
	msg   string
}{}

//line yaccpar:1

/*	parser for yacc output	*/

var (
	yyDebug        = 0
	yyErrorVerbose = false
)

type yyLexer interface {
	Lex(lval *yySymType) int
	Error(s string)
}

type yyParser interface {
	Parse(yyLexer) int
	Lookahead() int
}

type yyParserImpl struct {
	lval  yySymType
	stack [yyInitialStackSize]yySymType
	char  int
}

func (p *yyParserImpl) Lookahead() int {
	return p.char
}

func yyNewParser() yyParser {
	return &yyParserImpl{}
}

const yyFlag = -1000

func yyTokname(c int) string {
	if c >= 1 && c-1 < len(yyToknames) {
		if yyToknames[c-1] != "" {
			return yyToknames[c-1]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func yyStatname(s int) string {
	if s >= 0 && s < len(yyStatenames) {
		if yyStatenames[s] != "" {
			return yyStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func yyErrorMessage(state, lookAhead int) string {
	const TOKSTART = 4

	if !yyErrorVerbose {
		return "syntax error"
	}

	for _, e := range yyErrorMessages {
		if e.state == state && e.token == lookAhead {
			return "syntax error: " + e.msg
		}
	}

	res := "syntax error: unexpected " + yyTokname(lookAhead)

	// To match Bison, suggest at most four expected tokens.
	expected := make([]int, 0, 4)

	// Look for shiftable tokens.
	base := yyPact[state]
	for tok := TOKSTART; tok-1 < len(yyToknames); tok++ {
		if n := base + tok; n >= 0 && n < yyLast && yyChk[yyAct[n]] == tok {
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}
	}

	if yyDef[state] == -2 {
		i := 0
		for yyExca[i] != -1 || yyExca[i+1] != state {
			i += 2
		}

		// Look for tokens that we accept or reduce.
		for i += 2; yyExca[i] >= 0; i += 2 {
			tok := yyExca[i]
			if tok < TOKSTART || yyExca[i+1] == 0 {
				continue
			}
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}

		// If the default action is to accept or reduce, give up.
		if yyExca[i+1] != 0 {
			return res
		}
	}

	for i, tok := range expected {
		if i == 0 {
			res += ", expecting "
		} else {
			res += " or "
		}
		res += yyTokname(tok)
	}
	return res
}

func yylex1(lex yyLexer, lval *yySymType) (char, token int) {
	token = 0
	char = lex.Lex(lval)
	if char <= 0 {
		token = yyTok1[0]
		goto out
	}
	if char < len(yyTok1) {
		token = yyTok1[char]
		goto out
	}
	if char >= yyPrivate {
		if char < yyPrivate+len(yyTok2) {
			token = yyTok2[char-yyPrivate]
			goto out
		}
	}
	for i := 0; i < len(yyTok3); i += 2 {
		token = yyTok3[i+0]
		if token == char {
			token = yyTok3[i+1]
			goto out
		}
	}

out:
	if token == 0 {
		token = yyTok2[1] /* unknown char */
	}
	if yyDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", yyTokname(token), uint(char))
	}
	return char, token
}

func yyParse(yylex yyLexer) int {
	return yyNewParser().Parse(yylex)
}

func (yyrcvr *yyParserImpl) Parse(yylex yyLexer) int {
	var yyn int
	var yyVAL yySymType
	var yyDollar []yySymType
	_ = yyDollar // silence set and not used
	yyS := yyrcvr.stack[:]

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yystate := 0
	yyrcvr.char = -1
	yytoken := -1 // yyrcvr.char translated into internal numbering
	defer func() {
		// Make sure we report no lookahead when not parsing.
		yystate = -1
		yyrcvr.char = -1
		yytoken = -1
	}()
	yyp := -1
	goto yystack

ret0:
	return 0

ret1:
	return 1

yystack:
	/* put a state and value onto the stack */
	if yyDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", yyTokname(yytoken), yyStatname(yystate))
	}

	yyp++
	if yyp >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyS[yyp] = yyVAL
	yyS[yyp].yys = yystate

yynewstate:
	yyn = yyPact[yystate]
	if yyn <= yyFlag {
		goto yydefault /* simple state */
	}
	if yyrcvr.char < 0 {
		yyrcvr.char, yytoken = yylex1(yylex, &yyrcvr.lval)
	}
	yyn += yytoken
	if yyn < 0 || yyn >= yyLast {
		goto yydefault
	}
	yyn = yyAct[yyn]
	if yyChk[yyn] == yytoken { /* valid shift */
		yyrcvr.char = -1
		yytoken = -1
		yyVAL = yyrcvr.lval
		yystate = yyn
		if Errflag > 0 {
			Errflag--
		}
		goto yystack
	}

yydefault:
	/* default state action */
	yyn = yyDef[yystate]
	if yyn == -2 {
		if yyrcvr.char < 0 {
			yyrcvr.char, yytoken = yylex1(yylex, &yyrcvr.lval)
		}

		/* look through exception table */
		xi := 0
		for {
			if yyExca[xi+0] == -1 && yyExca[xi+1] == yystate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			yyn = yyExca[xi+0]
			if yyn < 0 || yyn == yytoken {
				break
			}
		}
		yyn = yyExca[xi+1]
		if yyn < 0 {
			goto ret0
		}
	}
	if yyn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			yylex.Error(yyErrorMessage(yystate, yytoken))
			Nerrs++
			if yyDebug >= 1 {
				__yyfmt__.Printf("%s", yyStatname(yystate))
				__yyfmt__.Printf(" saw %s\n", yyTokname(yytoken))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for yyp >= 0 {
				yyn = yyPact[yyS[yyp].yys] + yyErrCode
				if yyn >= 0 && yyn < yyLast {
					yystate = yyAct[yyn] /* simulate a shift of "error" */
					if yyChk[yystate] == yyErrCode {
						goto yystack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if yyDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", yyS[yyp].yys)
				}
				yyp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if yyDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", yyTokname(yytoken))
			}
			if yytoken == yyEofCode {
				goto ret1
			}
			yyrcvr.char = -1
			yytoken = -1
			goto yynewstate /* try again in the same state */
		}
	}

	/* reduction by production yyn */
	if yyDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", yyn, yyStatname(yystate))
	}

	yynt := yyn
	yypt := yyp
	_ = yypt // guard against "declared and not used"

	yyp -= yyR2[yyn]
	// yyp is now the index of $0. Perform the default action. Iff the
	// reduced production is ε, $1 is possibly out of range.
	if yyp+1 >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyVAL = yyS[yyp+1]

	/* consult goto table to find next state */
	yyn = yyR1[yyn]
	yyg := yyPgo[yyn]
	yyj := yyg + yyS[yyp].yys + 1

	if yyj >= yyLast {
		yystate = yyAct[yyg]
	} else {
		yystate = yyAct[yyj]
		if yyChk[yystate] != -yyn {
			yystate = yyAct[yyg]
		}
	}
	// dummy call; replaced with literal code
	switch yynt {

	case 1:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:193
		{
			yylex.(*input).file = &File{Stmt: yyDollar[1].exprs}
			return 0
		}
	case 2:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:200
		{
			statements := yyDollar[4].exprs
			if yyDollar[2].exprs != nil {
				// $2 can only contain *CommentBlock objects, each of them contains a non-empty After slice
				cb := yyDollar[2].exprs[len(yyDollar[2].exprs)-1].(*CommentBlock)
				// $4 can't be empty and can't start with a comment
				stmt := yyDollar[4].exprs[0]
				start, _ := stmt.Span()
				if start.Line-cb.After[len(cb.After)-1].Start.Line == 1 {
					// The first statement of $4 starts on the next line after the last comment of $2.
					// Attach the last comment to the first statement
					stmt.Comment().Before = cb.After
					yyDollar[2].exprs = yyDollar[2].exprs[:len(yyDollar[2].exprs)-1]
				}
				statements = append(yyDollar[2].exprs, yyDollar[4].exprs...)
			}
			yyVAL.exprs = statements
			yyVAL.lastStmt = yyDollar[4].lastStmt
		}
	case 3:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:220
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 6:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:228
		{
			yyVAL.exprs = nil
			yyVAL.lastStmt = nil
		}
	case 7:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:233
		{
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = yyDollar[1].lastStmt
			if yyVAL.lastStmt == nil {
				cb := &CommentBlock{Start: yyDollar[2].pos}
				yyVAL.exprs = append(yyVAL.exprs, cb)
				yyVAL.lastStmt = cb
			}
			com := yyVAL.lastStmt.Comment()
			com.After = append(com.After, Comment{Start: yyDollar[2].pos, Token: yyDollar[2].tok})
		}
	case 8:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:245
		{
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = nil
		}
	case 9:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:251
		{
			yyVAL.exprs = nil
			yyVAL.lastStmt = nil
		}
	case 10:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:256
		{
			// If this statement follows a comment block,
			// attach the comments to the statement.
			if cb, ok := yyDollar[1].lastStmt.(*CommentBlock); ok {
				yyVAL.exprs = append(yyDollar[1].exprs[:len(yyDollar[1].exprs)-1], yyDollar[2].exprs...)
				yyDollar[2].exprs[0].Comment().Before = cb.After
				yyVAL.lastStmt = yyDollar[2].lastStmt
				break
			}

			// Otherwise add to list.
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[2].exprs...)
			yyVAL.lastStmt = yyDollar[2].lastStmt

			// Consider this input:
			//
			//	foo()
			//	# bar
			//	baz()
			//
			// If we've just parsed baz(), the # bar is attached to
			// foo() as an After comment. Make it a Before comment
			// for baz() instead.
			if x := yyDollar[1].lastStmt; x != nil {
				com := x.Comment()
				// stmt is never empty
				yyDollar[2].exprs[0].Comment().Before = com.After
				com.After = nil
			}
		}
	case 11:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:287
		{
			// Blank line; sever last rule from future comments.
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = nil
		}
	case 12:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:293
		{
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = yyDollar[1].lastStmt
			if yyVAL.lastStmt == nil {
				cb := &CommentBlock{Start: yyDollar[2].pos}
				yyVAL.exprs = append(yyVAL.exprs, cb)
				yyVAL.lastStmt = cb
			}
			com := yyVAL.lastStmt.Comment()
			com.After = append(com.After, Comment{Start: yyDollar[2].pos, Token: yyDollar[2].tok})
		}
	case 13:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:307
		{
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = yyDollar[1].exprs[len(yyDollar[1].exprs)-1]
		}
	case 14:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:312
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
			yyVAL.lastStmt = yyDollar[1].expr
			if cbs := extractTrailingComments(yyDollar[1].expr); len(cbs) > 0 {
				yyVAL.exprs = append(yyVAL.exprs, cbs...)
				yyVAL.lastStmt = cbs[len(cbs)-1]
				if yyDollar[1].lastStmt == nil {
					yyVAL.lastStmt = nil
				}
			}
		}
	case 15:
		yyDollar = yyS[yypt-7 : yypt+1]
//line build/parse.y:326
		{
			yyVAL.expr = &DefStmt{
				Function: Function{
					StartPos: yyDollar[1].pos,
					Params:   yyDollar[4].exprs,
					Body:     yyDollar[7].exprs,
				},
				Name:           yyDollar[2].tok,
				ColonPos:       yyDollar[6].pos,
				ForceCompact:   forceCompact(yyDollar[3].pos, yyDollar[4].exprs, yyDollar[5].pos),
				ForceMultiLine: forceMultiLine(yyDollar[3].pos, yyDollar[4].exprs, yyDollar[5].pos),
			}
			yyVAL.lastStmt = yyDollar[7].lastStmt
		}
	case 16:
		yyDollar = yyS[yypt-6 : yypt+1]
//line build/parse.y:341
		{
			yyVAL.expr = &ForStmt{
				For:  yyDollar[1].pos,
				Vars: yyDollar[2].expr,
				X:    yyDollar[4].expr,
				Body: yyDollar[6].exprs,
			}
			yyVAL.lastStmt = yyDollar[6].lastStmt
		}
	case 17:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:351
		{
			yyVAL.expr = yyDollar[1].ifstmt
			yyVAL.lastStmt = yyDollar[1].lastStmt
		}
	case 18:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:359
		{
			yyVAL.ifstmt = &IfStmt{
				If:   yyDollar[1].pos,
				Cond: yyDollar[2].expr,
				True: yyDollar[4].exprs,
			}
			yyVAL.lastStmt = yyDollar[4].lastStmt
		}
	case 19:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:368
		{
			yyVAL.ifstmt = yyDollar[1].ifstmt
			inner := yyDollar[1].ifstmt
			for len(inner.False) == 1 {
				inner = inner.False[0].(*IfStmt)
			}
			inner.ElsePos = End{Pos: yyDollar[2].pos}
			inner.False = []Expr{
				&IfStmt{
					If:   yyDollar[2].pos,
					Cond: yyDollar[3].expr,
					True: yyDollar[5].exprs,
				},
			}
			yyVAL.lastStmt = yyDollar[5].lastStmt
		}
	case 21:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:389
		{
			yyVAL.ifstmt = yyDollar[1].ifstmt
			inner := yyDollar[1].ifstmt
			for len(inner.False) == 1 {
				inner = inner.False[0].(*IfStmt)
			}
			inner.ElsePos = End{Pos: yyDollar[2].pos}
			inner.False = yyDollar[4].exprs
			yyVAL.lastStmt = yyDollar[4].lastStmt
		}
	case 24:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:406
		{
			yyVAL.exprs = append([]Expr{yyDollar[1].expr}, yyDollar[2].exprs...)
			yyVAL.lastStmt = yyVAL.exprs[len(yyVAL.exprs)-1]
		}
	case 25:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:412
		{
			yyVAL.exprs = []Expr{}
		}
	case 26:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:416
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 28:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:423
		{
			yyVAL.expr = &ReturnStmt{
				Return: yyDollar[1].pos,
				Result: yyDollar[2].expr,
			}
		}
	case 29:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:430
		{
			yyVAL.expr = &ReturnStmt{
				Return: yyDollar[1].pos,
			}
		}
	case 30:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:435
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 31:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:436
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 32:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:438
		{
			yyVAL.expr = &BranchStmt{
				Token:    yyDollar[1].tok,
				TokenPos: yyDollar[1].pos,
			}
		}
	case 33:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:445
		{
			yyVAL.expr = &BranchStmt{
				Token:    yyDollar[1].tok,
				TokenPos: yyDollar[1].pos,
			}
		}
	case 34:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:452
		{
			yyVAL.expr = &BranchStmt{
				Token:    yyDollar[1].tok,
				TokenPos: yyDollar[1].pos,
			}
		}
	case 39:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:466
		{
			yyVAL.expr = yyDollar[1].string
		}
	case 40:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:470
		{
			yyVAL.expr = &DotExpr{
				X:       yyDollar[1].expr,
				Dot:     yyDollar[2].pos,
				NamePos: yyDollar[3].pos,
				Name:    yyDollar[3].tok,
			}
		}
	case 41:
		yyDollar = yyS[yypt-7 : yypt+1]
//line build/parse.y:479
		{
			load := &LoadStmt{
				Load:         yyDollar[1].pos,
				Module:       yyDollar[3].string,
				Rparen:       End{Pos: yyDollar[7].pos},
				ForceCompact: yyDollar[1].pos.Line == yyDollar[7].pos.Line,
			}
			for _, arg := range yyDollar[5].loadargs {
				load.From = append(load.From, &arg.from)
				load.To = append(load.To, &arg.to)
			}
			yyVAL.expr = load
		}
	case 42:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:493
		{
			yyVAL.expr = &CallExpr{
				X:              yyDollar[1].expr,
				ListStart:      yyDollar[2].pos,
				List:           yyDollar[3].exprs,
				End:            End{Pos: yyDollar[4].pos},
				ForceCompact:   forceCompact(yyDollar[2].pos, yyDollar[3].exprs, yyDollar[4].pos),
				ForceMultiLine: forceMultiLine(yyDollar[2].pos, yyDollar[3].exprs, yyDollar[4].pos),
			}
		}
	case 43:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:504
		{
			yyVAL.expr = &IndexExpr{
				X:          yyDollar[1].expr,
				IndexStart: yyDollar[2].pos,
				Y:          yyDollar[3].expr,
				End:        yyDollar[4].pos,
			}
		}
	case 44:
		yyDollar = yyS[yypt-6 : yypt+1]
//line build/parse.y:513
		{
			yyVAL.expr = &SliceExpr{
				X:          yyDollar[1].expr,
				SliceStart: yyDollar[2].pos,
				From:       yyDollar[3].expr,
				FirstColon: yyDollar[4].pos,
				To:         yyDollar[5].expr,
				End:        yyDollar[6].pos,
			}
		}
	case 45:
		yyDollar = yyS[yypt-8 : yypt+1]
//line build/parse.y:524
		{
			yyVAL.expr = &SliceExpr{
				X:           yyDollar[1].expr,
				SliceStart:  yyDollar[2].pos,
				From:        yyDollar[3].expr,
				FirstColon:  yyDollar[4].pos,
				To:          yyDollar[5].expr,
				SecondColon: yyDollar[6].pos,
				Step:        yyDollar[7].expr,
				End:         yyDollar[8].pos,
			}
		}
	case 46:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:537
		{
			yyVAL.expr = &ListExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[3].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
			}
		}
	case 47:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:546
		{
			yyVAL.expr = &Comprehension{
				Curly:          false,
				Lbrack:         yyDollar[1].pos,
				Body:           yyDollar[2].expr,
				Clauses:        yyDollar[3].exprs,
				End:            End{Pos: yyDollar[4].pos},
				ForceMultiLine: forceMultiLineComprehension(yyDollar[1].pos, yyDollar[2].expr, yyDollar[3].exprs, yyDollar[4].pos),
			}
		}
	case 48:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:557
		{
			yyVAL.expr = &Comprehension{
				Curly:          true,
				Lbrack:         yyDollar[1].pos,
				Body:           yyDollar[2].kv,
				Clauses:        yyDollar[3].exprs,
				End:            End{Pos: yyDollar[4].pos},
				ForceMultiLine: forceMultiLineComprehension(yyDollar[1].pos, yyDollar[2].kv, yyDollar[3].exprs, yyDollar[4].pos),
			}
		}
	case 49:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:568
		{
			exprValues := make([]Expr, 0, len(yyDollar[2].kvs))
			for _, kv := range yyDollar[2].kvs {
				exprValues = append(exprValues, Expr(kv))
			}
			yyVAL.expr = &DictExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].kvs,
				End:            End{Pos: yyDollar[3].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, exprValues, yyDollar[3].pos),
			}
		}
	case 50:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:581
		{
			yyVAL.expr = &SetExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[4].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[4].pos),
			}
		}
	case 51:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:590
		{
			if len(yyDollar[2].exprs) == 1 && yyDollar[2].comma.Line == 0 {
				// Just a parenthesized expression, not a tuple.
				yyVAL.expr = &ParenExpr{
					Start:          yyDollar[1].pos,
					X:              yyDollar[2].exprs[0],
					End:            End{Pos: yyDollar[3].pos},
					ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
				}
			} else {
				yyVAL.expr = &TupleExpr{
					Start:          yyDollar[1].pos,
					List:           yyDollar[2].exprs,
					End:            End{Pos: yyDollar[3].pos},
					ForceCompact:   forceCompact(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
					ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
				}
			}
		}
	case 52:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:611
		{
			yyVAL.exprs = nil
		}
	case 53:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:615
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 54:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:621
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 55:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:625
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 57:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:632
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 58:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:636
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 59:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:640
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 60:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:645
		{
			yyVAL.loadargs = []*struct {
				from Ident
				to   Ident
			}{yyDollar[1].loadarg}
		}
	case 61:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:649
		{
			yyDollar[1].loadargs = append(yyDollar[1].loadargs, yyDollar[3].loadarg)
			yyVAL.loadargs = yyDollar[1].loadargs
		}
	case 62:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:655
		{
			start := yyDollar[1].string.Start.add("'")
			if yyDollar[1].string.TripleQuote {
				start = start.add("''")
			}
			yyVAL.loadarg = &struct {
				from Ident
				to   Ident
			}{
				from: Ident{
					Name:    yyDollar[1].string.Value,
					NamePos: start,
				},
				to: Ident{
					Name:    yyDollar[1].string.Value,
					NamePos: start,
				},
			}
		}
	case 63:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:672
		{
			start := yyDollar[3].string.Start.add("'")
			if yyDollar[3].string.TripleQuote {
				start = start.add("''")
			}
			yyVAL.loadarg = &struct {
				from Ident
				to   Ident
			}{
				from: Ident{
					Name:    yyDollar[3].string.Value,
					NamePos: start,
				},
				to: *yyDollar[1].expr.(*Ident),
			}
		}
	case 64:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:687
		{
			yyVAL.exprs = nil
		}
	case 65:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:691
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 66:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:697
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 67:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:701
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 69:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:708
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 70:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:712
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 71:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:716
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, nil)
		}
	case 72:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:720
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 74:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:727
		{
			tuple, ok := yyDollar[1].expr.(*TupleExpr)
			if !ok || !tuple.NoBrackets {
				tuple = &TupleExpr{
					List:           []Expr{yyDollar[1].expr},
					NoBrackets:     true,
					ForceCompact:   true,
					ForceMultiLine: false,
				}
			}
			tuple.List = append(tuple.List, yyDollar[3].expr)
			yyVAL.expr = tuple
		}
	case 75:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:742
		{
			yyVAL.expr = nil
		}
	case 78:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:750
		{
			yyVAL.expr = &LambdaExpr{
				Function: Function{
					StartPos: yyDollar[1].pos,
					Params:   yyDollar[2].exprs,
					Body:     []Expr{yyDollar[4].expr},
				},
			}
		}
	case 79:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:759
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 80:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:760
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 81:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:761
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 82:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:762
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 83:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:763
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 84:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:764
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 85:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:765
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 86:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:766
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 87:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:767
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 88:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:768
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 89:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:769
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 90:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:770
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 91:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:771
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 92:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:772
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 93:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:773
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 94:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:774
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 95:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:775
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 96:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:776
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, "not in", yyDollar[4].expr)
		}
	case 97:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:777
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 98:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:778
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 99:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:779
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 100:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:780
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 101:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:781
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 102:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:782
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 103:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:783
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 104:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:785
		{
			if b, ok := yyDollar[3].expr.(*UnaryExpr); ok && b.Op == "not" {
				yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, "is not", b.X)
			} else {
				yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
			}
		}
	case 105:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:793
		{
			yyVAL.expr = &ConditionalExpr{
				Then:      yyDollar[1].expr,
				IfStart:   yyDollar[2].pos,
				Test:      yyDollar[3].expr,
				ElseStart: yyDollar[4].pos,
				Else:      yyDollar[5].expr,
			}
		}
	case 106:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:805
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 107:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:809
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 108:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:814
		{
			yyVAL.expr = nil
		}
	case 110:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:820
		{
			yyVAL.exprs, yyVAL.comma = nil, Position{}
		}
	case 111:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:824
		{
			yyVAL.exprs, yyVAL.comma = yyDollar[1].exprs, yyDollar[2].pos
		}
	case 112:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:833
		{
			yyVAL.pos = Position{}
		}
	case 114:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:839
		{
			yyVAL.kv = &KeyValueExpr{
				Key:   yyDollar[1].expr,
				Colon: yyDollar[2].pos,
				Value: yyDollar[3].expr,
			}
		}
	case 115:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:849
		{
			yyVAL.kvs = []*KeyValueExpr{yyDollar[1].kv}
		}
	case 116:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:853
		{
			yyVAL.kvs = append(yyDollar[1].kvs, yyDollar[3].kv)
		}
	case 117:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:858
		{
			yyVAL.kvs = nil
		}
	case 118:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:862
		{
			yyVAL.kvs = yyDollar[1].kvs
		}
	case 119:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:866
		{
			yyVAL.kvs = yyDollar[1].kvs
		}
	case 121:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:873
		{
			tuple, ok := yyDollar[1].expr.(*TupleExpr)
			if !ok || !tuple.NoBrackets {
				tuple = &TupleExpr{
					List:           []Expr{yyDollar[1].expr},
					NoBrackets:     true,
					ForceCompact:   true,
					ForceMultiLine: false,
				}
			}
			tuple.List = append(tuple.List, yyDollar[3].expr)
			yyVAL.expr = tuple
		}
	case 122:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:889
		{
			yyVAL.string = &StringExpr{
				Start:       yyDollar[1].pos,
				Value:       yyDollar[1].str,
				TripleQuote: yyDollar[1].triple,
				End:         yyDollar[1].pos.add(yyDollar[1].tok),
				Token:       yyDollar[1].tok,
			}
		}
	case 123:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:901
		{
			yyVAL.expr = &Ident{NamePos: yyDollar[1].pos, Name: yyDollar[1].tok}
		}
	case 124:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:907
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: yyDollar[1].tok + "." + yyDollar[3].tok}
		}
	case 125:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:911
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: yyDollar[1].tok + "."}
		}
	case 126:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:915
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: "." + yyDollar[2].tok}
		}
	case 127:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:919
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: yyDollar[1].tok}
		}
	case 128:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:925
		{
			yyVAL.expr = &ForClause{
				For:  yyDollar[1].pos,
				Vars: yyDollar[2].expr,
				In:   yyDollar[3].pos,
				X:    yyDollar[4].expr,
			}
		}
	case 129:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:936
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 130:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:940
		{
			yyVAL.exprs = append(yyDollar[1].exprs, &IfClause{
				If:   yyDollar[2].pos,
				Cond: yyDollar[3].expr,
			})
		}
	case 131:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:949
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 132:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:953
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[2].exprs...)
		}
	}
	goto yystack /* stack new state and value */
}
