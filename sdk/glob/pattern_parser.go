package glob

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"
	"unicode"

	"github.com/pkg/errors"
)

type Token int

const (
	ILLEGAL Token = iota
	EOF
	SLASH
	LITERAL
	STAR
	SQUARE_BRACKET_LEFT
	SQUARE_BRACKET_RIGHT
	INTEROGATION_MARK
	DOUBLESTAR
	SQUARE_BRACKET
	DOUBLESTARSLASH
)

const (
	eof     = rune(0)
	nothing = rune(32)
)

func isEOF(ch rune) bool {
	return ch == eof
}

func isPattern(ch rune) bool {
	return !isEOF(ch) && !isSlash(ch) && unicode.IsPrint(ch)
}

func isSlash(ch rune) bool {
	return ch == '/'
}

func isStar(ch rune) bool {
	return ch == '*'
}

func isInterogationMark(ch rune) bool {
	return ch == '?'
}

func isSquareBracketLeft(ch rune) bool {
	return ch == '['
}

func isSquareBracketRight(ch rune) bool {
	return ch == ']'
}

type pattern struct {
	raw       string
	isExclude bool
}

func (f *pattern) Match(s string) (string, error) {
	parser, err := f.newParser()
	if err != nil {
		return "", err
	}
	return parser.parseAndMatch(s)
}

type innerScanner struct {
	f *pattern
	r *bufio.Reader
	n int64
}

func (f *pattern) newScanner() *innerScanner {
	return &innerScanner{f: f, r: bufio.NewReader(strings.NewReader(f.raw))}
}
func (s *innerScanner) clone() *innerScanner {
	pattern := &pattern{raw: s.f.raw[s.n-1:]}
	return &innerScanner{
		f: pattern,
		r: bufio.NewReader(strings.NewReader(pattern.raw)),
	}
}

func (s *innerScanner) read() rune {
	ch, _, err := s.r.ReadRune()
	s.n++
	if err != nil {
		return eof
	}
	return ch
}
func (s *innerScanner) unread() {
	_ = s.r.UnreadRune()
	s.n--
}

func (s *innerScanner) scan() (tok Token, lit string) {
	ch := s.read()
	switch {
	case isEOF(ch):
		return EOF, ""

	case isInterogationMark(ch):
		s.unread()
		return s.scanInterogationMark()

	case isSlash(ch):
		s.unread()
		return s.scanSlash()

	case isStar(ch): // matches ** and **/ too
		s.unread()
		return s.scanStar()

	case isSquareBracketLeft(ch):
		s.unread()
		return s.scanSquareBracket()

	default:
		s.unread()
		return s.scanLiteral()
	}
}
func (s *innerScanner) scanLiteral() (tok Token, lit string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())
	return LITERAL, buf.String()
}
func (s *innerScanner) scanInterogationMark() (tok Token, lit string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())
	return INTEROGATION_MARK, buf.String()
}
func (s *innerScanner) scanStar() (tok Token, lit string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())
	nextCh := s.read()
	s.unread()
	if isStar(nextCh) {
		return s.scanDoubleStar()
	}
	return STAR, buf.String()
}

func (s *innerScanner) scanDoubleStar() (tok Token, lit string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())
	nextCh := s.read()
	if isSlash(nextCh) {
		return DOUBLESTARSLASH, "**/"
	} else {
		s.unread()
	}
	return DOUBLESTAR, buf.String()
}

func (s *innerScanner) scanSquareBracket() (tok Token, lit string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())
	for {
		ch := s.read()
		if isEOF(ch) || isStar(ch) || isSquareBracketLeft(ch) || isInterogationMark(ch) {
			s.unread()
			return LITERAL, buf.String()
		}
		buf.WriteRune(ch)
		if isSquareBracketRight(ch) {
			break
		}
	}
	return SQUARE_BRACKET, buf.String()
}
func (s *innerScanner) scanSlash() (tok Token, lit string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())
	return SLASH, buf.String()
}

type innerParser struct {
	s   *innerScanner
	buf struct {
		tok Token  // last read token
		lit string // last read literal
		n   int    // buffer size (max=1)
	}
}

func (f *pattern) newParser() (*innerParser, error) {
	return &innerParser{
		s: f.newScanner(),
	}, nil
}

func (p *innerParser) clone() *innerParser {
	return &innerParser{
		s: p.s.clone(),
	}
}

func (p *innerParser) scan() (tok Token, lit string) {
	// If we have a token on the buffer, then return it.
	if p.buf.n != 0 {
		p.buf.n = 0
		return p.buf.tok, p.buf.lit
	}
	// Otherwise read the next token from the scanner.
	tok, lit = p.s.scan()
	// Save it to the buffer in case we unscan later.
	p.buf.tok, p.buf.lit = tok, lit
	return
}

func (p *innerParser) unscan() { p.buf.n = 1 }

func (p *innerParser) parseAndMatch(s string) (result string, err error) {
	Debug("# PARSE AND MATCH: %s with %s", s, p.s.f.raw)
	defer func() {
		Debug("# Result :%s Error: %v", result, err)
	}()

	contentParser, err := newContentParser(s)
	if err != nil {
		return "", err
	}

	buf := new(strings.Builder)

	var inWildcard bool
	for {
		tok, lit := p.scan()
		Debug("## current token: %v %v", tok, lit)
		nextToken := EOF
		nextLit := ""
		if tok != EOF {
			nextToken, nextLit = p.scan()
			p.unscan()
		}

		Debug("## checking %q (next:%q)", lit, nextLit)
		Debug("## current buffer: %q", buf.String())

		switch tok {
		case EOF:
			contentToken, _ := contentParser.scan()
			if contentToken == EOF { // the pattern is over and the content too. It's a success
				return buf.String(), nil
			}
			return "", nil
		case SLASH:
			contentToken, contentLit := contentParser.scan()
			Debug("### / current buffer: %q, read content: [%v] %q, nextToken: [%v] %q", buf.String(), contentToken, contentLit, nextToken, nextLit)
			if contentToken == EOF {
				continue
			}
			if !inWildcard {
				buf.Reset() // reset the buff
			} else {
				buf.WriteString(lit)
				if contentToken != SLASH {
					contentParser.unscan()
				}
			}
		case DOUBLESTAR, DOUBLESTARSLASH:
			var contentParserIndex = contentParser.index
			var lastSlashIndex = -1
			var index = -1
			var accumulator = new(strings.Builder)
			var stopGLobing bool
			inWildcard = true
			for !stopGLobing {
				index++
				contentToken, contentLit := contentParser.scan()
				Debug("### ** current buffer: %q, read content: [%v] %q, nextToken: [%v] %q", buf.String(), contentToken, contentLit, nextToken, nextLit)
				if contentToken == EOF {
					break
				}

				if tok == DOUBLESTARSLASH && nextToken != STAR && contentToken == SLASH {
					stopGLobing = true
				}

				if contentToken == LITERAL {
					accumulator.WriteString(contentLit)
					continue
				} else if contentToken == SLASH && nextToken != EOF {
					lastSlashIndex = index
					accumulator.WriteString(contentLit)
					continue
				} else if nextToken == EOF {
					accumulator.WriteString(contentLit)
					continue
				}
				Debug("### exiting globstar")
				break
			}
			if lastSlashIndex > -1 {
				str := accumulator.String()
				str = str[:lastSlashIndex]
				Debug("### str: %q", str)
				Debug("### rewindTo: %d", contentParserIndex+lastSlashIndex)
				contentParser.rewindTo(contentParserIndex + lastSlashIndex)
				buf.WriteString(str)
				if tok == DOUBLESTARSLASH {
					buf.WriteRune('/')
					contentParser.scan()
				}
			} else {
				buf.WriteString(accumulator.String())
			}
			Debug("### buffer at the end of DOUBLESTAR: %q", buf.String())
		case STAR:
			// Star will consume the content until the next '/' of EOF
			inWildcard = true
			for {
				contentToken, contentLit := contentParser.scan()
				Debug("### * current buffer: %q, read content: [%v] %q, nextToken: [%v] %q", buf.String(), contentToken, contentLit, nextToken, nextLit)
				if contentToken == EOF {
					break
				}
				if contentToken == LITERAL && contentLit == nextLit {
					Debug("### STAR leftover: %s", contentParser.leftover())
					// Try to match more with STAR
					subP := p.clone()
					subRes, err := subP.parseAndMatch(contentLit + contentParser.leftover())
					if err != nil {
						Debug("### Sub STAR parser error: %v", err)
						contentParser.unscan()
						break
					}
					if len(subRes) > 1 {
						Debug("### Sub STAR found some stuff: %q", subRes)
						for i := 0; i < len(subRes); i++ {
							contentParser.scan()
						}
						buf.WriteString(subRes)
						for i := 0; i < int(p.s.n); i++ {
							tt, tl := p.scan()
							Debug("### Sub STAR has consumed token %q", tl)
							if tt == EOF {
								break
							}
						}
						continue
					}
				}
				if contentToken != SLASH {
					buf.WriteString(contentLit)
					continue
				}
				break
			}
		case INTEROGATION_MARK:
			contentToken, contentLit := contentParser.scan()
			Debug("### ? current buffer: %q, read content: [%v] %q, nextToken: [%v] %q", buf.String(), contentToken, contentLit, nextToken, nextLit)

			if contentToken == EOF { // the pattern token is a litteral but there is nothing left in the content parser
				return "", nil
			}
			if contentToken != LITERAL { // the pattern token is a ? but the next token from the content is not a literral
				return "", nil
			}
			buf.WriteString(contentLit)
		case LITERAL:
			contentToken, contentLit := contentParser.scan()
			Debug("### L current buffer: %q, read content: [%v] %q, nextToken: [%v] %q", buf.String(), contentToken, contentLit, nextToken, nextLit)

			if contentToken == EOF { // the pattern token is a litteral but there is nothing left in the content parser
				return "", nil
			}
			if contentToken == SLASH {
				contentToken, contentLit = contentParser.scan()
				Debug("### LL current buffer: %q, read content: [%v] %q, nextToken: [%v] %q", buf.String(), contentToken, contentLit, nextToken, nextLit)
			}

			if lit != contentLit { // the literal from the pattern doesn't match with the literal from the content
				Debug("### out")
				return "", nil
			}
			buf.WriteString(contentLit)
		case SQUARE_BRACKET:
			rgp, err := regexp.Compile(lit)
			if err != nil {
				return "", errors.Wrapf(err, "unexpected %q", lit)
			}
			contentToken, contentLit := contentParser.scan()
			Debug("### [] current buffer: %q, read content: [%v] %q, nextToken: [%v] %q", buf.String(), contentToken, contentLit, nextToken, nextLit)

			if contentToken != LITERAL {
				return "", nil
			}
			if !rgp.MatchString(contentLit) {
				return "", nil
			}
			buf.WriteString(contentLit)
		}
	}
}

type contentScanner struct {
	r *bufio.Reader
}

func newContentScanner(s string) *contentScanner {
	return &contentScanner{bufio.NewReader(strings.NewReader(s))}
}
func (s *contentScanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	return ch
}
func (s *contentScanner) unread() { _ = s.r.UnreadRune() }

func (s *contentScanner) scan() (tok Token, lit string) {
	ch := s.read()
	switch {
	case isEOF(ch):
		return EOF, ""
	case isSlash(ch):
		s.unread()
		return s.scanSlash()
	default:
		s.unread()
		return s.scanLiteral()
	}
}
func (s *contentScanner) scanLiteral() (tok Token, lit string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())
	return LITERAL, buf.String()
}
func (s *contentScanner) scanSlash() (tok Token, lit string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())
	return SLASH, buf.String()
}

type contentParser struct {
	index   int
	raw     string
	scanner *contentScanner
	buf     struct {
		tok Token  // last read token
		lit string // last read literal
		n   int    // buffer size (max=1)
	}
}

func newContentParser(s string) (*contentParser, error) {
	return &contentParser{
		raw:     s,
		scanner: newContentScanner(s),
	}, nil
}
func (p *contentParser) scan() (tok Token, lit string) {
	p.index++
	// If we have a token on the buffer, then return it.
	if p.buf.n != 0 {
		p.buf.n = 0
		return p.buf.tok, p.buf.lit
	}
	// Otherwise read the next token from the scanner.
	tok, lit = p.scanner.scan()
	// Save it to the buffer in case we unscan later.
	p.buf.tok, p.buf.lit = tok, lit
	return
}

func (p *contentParser) unscan() {
	p.index--
	p.buf.n = 1
}

func (p *contentParser) rewindTo(r int) {
	p.index = 0
	p.scanner = newContentScanner(p.raw)
	for i := 0; i < r; i++ {
		p.scan()
	}
}

func (p *contentParser) leftover() string {
	return p.raw[p.index:]
}
