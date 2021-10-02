package index

import (
	"errors"
	"math"
	"unicode/utf8"
)

var (
	ErrExpectedIdentifier   = errors.New("expected field identifier")
	ErrExpectedColon        = errors.New("expected ':'")
	ErrExpectedSpace        = errors.New("expected ' '")
	ErrExpectedNumber       = errors.New("expected signed integer")
	ErrExpectedMinusOrSpace = errors.New("expected '-' or ' '")
	ErrExpectedDirection    = errors.New("expected '<' or '>'")
	ErrExpectedString       = errors.New("expected string")
)

type FieldType int

const (
	FTString FieldType = iota
	FTExactString
	FTInt
	FTRange
)

type Field struct {
	Name       string
	Type       FieldType
	String     string
	Int1, Int2 int32
	Value      interface{}
}

type Parser struct {
	data               []byte
	current            rune
	progress, previous int
	result             []Field
}

func (p *Parser) Parse(data []byte) ([]Field, int, error) {
	p.data = data
	p.progress = 0
	p.previous = 0
	p.result = p.result[:0]

	for p.Advance() {
		if !IsIdentStart(p.current) {
			return nil, p.progress, ErrExpectedIdentifier
		}

		name := p.Ident()

		if p.current != ':' {
			return nil, p.progress, ErrExpectedColon
		}
		p.Advance()

		if p.current != ' ' {
			return nil, p.progress, ErrExpectedSpace
		}
		p.Advance()

		stringType := FTString
		if p.current == '!' {
			stringType = FTExactString
			p.Advance()
		}

		switch p.current {
		case '"': // string
			p.result = append(p.result, Field{
				Name:   name,
				Type:   stringType,
				String: p.String(),
			})
		default: // int / range / identifier (string)

			if IsIdentStart(p.current) {
				p.result = append(p.result, Field{
					Name:   name,
					Type:   stringType,
					String: p.Ident(),
				})
				continue
			}

			if stringType == FTExactString {
				return nil, p.progress, ErrExpectedString
			}

			if min, ok := p.Number(); ok {
				switch p.current {
				case '-':
					p.Advance()
					if max, ok := p.Number(); ok {
						p.result = append(p.result, Field{
							Name: name,
							Type: FTRange,
							Int1: min,
							Int2: max,
						})
					} else {
						return nil, p.progress, ErrExpectedNumber
					}
				case ' ', utf8.RuneError:
					p.result = append(p.result, Field{
						Name: name,
						Type: FTInt,
						Int1: min,
					})
				default:
					return nil, p.progress, ErrExpectedMinusOrSpace
				}
				continue
			}

			var left bool
			switch p.current {
			case '>':
			case '<':
				left = true
			default:
				return nil, p.progress, ErrExpectedDirection
			}
			p.Advance()

			if num, ok := p.Number(); ok {
				field := Field{
					Name: name,
					Type: FTRange,
				}

				if left {
					field.Int1 = math.MinInt32
					field.Int2 = num
				} else {
					field.Int1 = num
					field.Int2 = math.MaxInt32
				}

				p.result = append(p.result, field)
			} else {
				return nil, p.progress, ErrExpectedNumber
			}
		}

	}

	return p.result, p.progress, nil
}

func (p *Parser) Number() (int32, bool) {
	var negative bool
	if p.current == '-' {
		negative = true
		p.Advance()
	}

	var result int32
	var hasNumber bool
	for IsNumber(p.current) {
		result *= 10
		result += int32(p.current - '0')
		hasNumber = true
		p.Advance()
	}

	if negative {
		result *= -1
	}

	return result, hasNumber
}

func (p *Parser) String() string {
	start := p.progress
	var escaped bool
	for p.Advance() && (p.current != '"' || escaped) {
		escaped = p.current == '\\'
	}
	end := p.previous
	p.Advance()
	return string(p.data[start:end])
}

func (p *Parser) Ident() string {
	start := p.previous
	for p.Advance() && IsIdent(p.current) {
	}
	return string(p.data[start:p.previous])
}

func (p *Parser) Advance() bool {
	r, size := utf8.DecodeRune(p.data[p.progress:])
	p.previous = p.progress
	p.progress += size
	p.current = r
	return r != utf8.RuneError
}

func IsIdent(r rune) bool {
	return IsIdentStart(r) || IsNumber(r)
}

func IsNumber(r rune) bool {
	return r >= '0' && r <= '9'
}

func IsIdentStart(r rune) bool {
	return r >= 'a' && r <= 'z' ||
		r >= 'A' && r <= 'Z' ||
		r == '_'
}
