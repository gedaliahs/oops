package detect

// Tokenize splits a shell command string into tokens, handling single quotes,
// double quotes, backslash escapes, and basic operators.
func Tokenize(input string) []string {
	var tokens []string
	var current []byte
	i := 0
	n := len(input)

	for i < n {
		ch := input[i]

		switch {
		case ch == '\\' && i+1 < n:
			// Backslash escape — take next char literally
			i++
			current = append(current, input[i])
			i++

		case ch == '\'':
			// Single-quoted string — no escaping inside
			i++ // skip opening quote
			for i < n && input[i] != '\'' {
				current = append(current, input[i])
				i++
			}
			if i < n {
				i++ // skip closing quote
			}

		case ch == '"':
			// Double-quoted string — backslash escapes work
			i++ // skip opening quote
			for i < n && input[i] != '"' {
				if input[i] == '\\' && i+1 < n {
					i++
					current = append(current, input[i])
				} else {
					current = append(current, input[i])
				}
				i++
			}
			if i < n {
				i++ // skip closing quote
			}

		case ch == ' ' || ch == '\t':
			if len(current) > 0 {
				tokens = append(tokens, string(current))
				current = current[:0]
			}
			i++

		case ch == '|' || ch == ';' || ch == '&':
			// Emit current token if any
			if len(current) > 0 {
				tokens = append(tokens, string(current))
				current = current[:0]
			}
			// Multi-char operators: ||, &&, |&
			if i+1 < n && ((ch == '|' && input[i+1] == '|') || (ch == '&' && input[i+1] == '&') || (ch == '|' && input[i+1] == '&')) {
				tokens = append(tokens, string(input[i:i+2]))
				i += 2
			} else {
				tokens = append(tokens, string(ch))
				i++
			}

		case ch == '>' || ch == '<':
			if len(current) > 0 {
				tokens = append(tokens, string(current))
				current = current[:0]
			}
			// Handle >>, <<, >&, 2>, 2>>
			if i+1 < n && input[i+1] == ch {
				tokens = append(tokens, string(input[i:i+2]))
				i += 2
			} else if ch == '>' && i+1 < n && input[i+1] == '&' {
				tokens = append(tokens, ">&")
				i += 2
			} else {
				tokens = append(tokens, string(ch))
				i++
			}

		case ch == '(' || ch == ')':
			if len(current) > 0 {
				tokens = append(tokens, string(current))
				current = current[:0]
			}
			tokens = append(tokens, string(ch))
			i++

		default:
			// Handle fd redirects like 2> or 2>>
			if ch >= '0' && ch <= '9' && i+1 < n && (input[i+1] == '>' || input[i+1] == '<') {
				if len(current) == 0 {
					if i+2 < n && input[i+1] == '>' && input[i+2] == '>' {
						tokens = append(tokens, string(input[i:i+3]))
						i += 3
						continue
					}
					tokens = append(tokens, string(input[i:i+2]))
					i += 2
					continue
				}
			}
			current = append(current, ch)
			i++
		}
	}

	if len(current) > 0 {
		tokens = append(tokens, string(current))
	}

	return tokens
}

// SplitPipeline splits a token list at pipe operators (|), returning
// segments. Each segment is the tokens for one command in the pipeline.
func SplitPipeline(tokens []string) [][]string {
	var segments [][]string
	var current []string
	for _, t := range tokens {
		if t == "|" || t == "|&" {
			if len(current) > 0 {
				segments = append(segments, current)
			}
			current = nil
		} else {
			current = append(current, t)
		}
	}
	if len(current) > 0 {
		segments = append(segments, current)
	}
	return segments
}

// SplitCommands splits tokens at command separators (;, &&, ||), returning
// each command segment.
func SplitCommands(tokens []string) [][]string {
	var segments [][]string
	var current []string
	for _, t := range tokens {
		if t == ";" || t == "&&" || t == "||" {
			if len(current) > 0 {
				segments = append(segments, current)
			}
			current = nil
		} else {
			current = append(current, t)
		}
	}
	if len(current) > 0 {
		segments = append(segments, current)
	}
	return segments
}
