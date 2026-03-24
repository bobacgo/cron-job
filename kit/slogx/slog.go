package slogx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Config holds logger setup options.
type Config struct {
	Level      slog.Leveler
	Writer     io.Writer
	AddSource  bool
	Color      bool
	Format     Format
	SourceRoot string

	colorSet bool
}

// Format indicates output format.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Option customizes logger Config.
type Option func(*Config)

// WithLevel sets the log level.
func WithLevel(level slog.Leveler) Option {
	return func(c *Config) {
		c.Level = level
	}
}

// WithWriter sets the output writer.
func WithWriter(w io.Writer) Option {
	return func(c *Config) {
		if w != nil {
			c.Writer = w
		}
	}
}

// WithFormat sets output format (text/json).
func WithFormat(format Format) Option {
	return func(c *Config) {
		if format != "" {
			c.Format = format
		}
	}
}

// WithColor explicitly enables or disables colorized output.
func WithColor(enabled bool) Option {
	return func(c *Config) {
		c.Color = enabled
		c.colorSet = true
	}
}

// WithSource controls whether to emit source location.
func WithSource(enabled bool) Option {
	return func(c *Config) {
		c.AddSource = enabled
	}
}

// LevelFatal defines a custom fatal level above error.
// slog 标准库没有 Fatal 级别，这里扩展一个更高的级别用于致命错误。
const LevelFatal = slog.Level(12)

// Fatal 以致命级别输出日志后直接退出进程（exit code 1）。
func Fatal(ctx context.Context, msg string, args ...any) {
	logWithSource(ctx, LevelFatal, msg, args, 1)
}

// Init builds a slog Logger with colorized console output and source location,
// sets it as the default logger, and returns it.
func Init(opts ...Option) *slog.Logger {
	cfg := &Config{
		Level:      slog.LevelInfo,
		Writer:     os.Stdout,
		AddSource:  true,
		Format:     FormatText,
		SourceRoot: defaultSourceRoot(),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if !cfg.colorSet {
		cfg.Color = shouldUseColor(cfg.Writer)
	}

	baseHandler := newBaseHandler(cfg)
	handler := &contextHandler{Handler: baseHandler}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func newBaseHandler(cfg *Config) slog.Handler {
	options := &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: cfg.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			return replaceAttr(cfg.Color, cfg.SourceRoot, a)
		},
	}

	if cfg.Format == FormatJSON {
		cfg.Color = false
		return slog.NewJSONHandler(cfg.Writer, options)
	}

	return &consoleHandler{
		w:           cfg.Writer,
		level:       cfg.Level,
		addSource:   cfg.AddSource,
		replaceAttr: options.ReplaceAttr,
		color:       cfg.Color,
		sourceRoot:  cfg.SourceRoot,
	}
}

func replaceAttr(enableColor bool, sourceRoot string, a slog.Attr) slog.Attr {
	switch a.Key {
	case slog.LevelKey:
		level, ok := valueToLevel(a.Value)
		if !ok {
			return a
		}
		upper := levelText(level)
		if enableColor {
			upper = colorize(level, upper)
		}
		a.Value = slog.StringValue(upper)
	case slog.TimeKey:
		if t, ok := valueToTime(a.Value); ok {
			a.Value = slog.StringValue(t.Local().Format("2006-01-02 15:04:05.000"))
		}
	case slog.SourceKey:
		if src, ok := a.Value.Any().(slog.Source); ok {
			file := trimSourcePath(sourceRoot, src.File)
			a.Value = slog.StringValue(fmt.Sprintf("%s:%d", file, src.Line))
		}
	}
	return a
}

func valueToLevel(v slog.Value) (slog.Level, bool) {
	switch v.Kind() {
	case slog.KindInt64:
		return slog.Level(v.Int64()), true
	case slog.KindString:
		return parseLevelValue(v.String())
	default:
		if lv, ok := v.Any().(slog.Level); ok {
			return lv, true
		}
		return slog.LevelInfo, false
	}
}

func parseLevelValue(s string) (slog.Level, bool) {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug, true
	case "info":
		return slog.LevelInfo, true
	case "warn", "warning":
		return slog.LevelWarn, true
	case "error", "err":
		return slog.LevelError, true
	case "fatal", "crit", "critical":
		return LevelFatal, true
	default:
		return slog.LevelInfo, false
	}
}

func valueToTime(v slog.Value) (time.Time, bool) {
	switch v.Kind() {
	case slog.KindTime:
		return v.Time(), true
	default:
		if t, ok := v.Any().(time.Time); ok {
			return t, true
		}
		return time.Time{}, false
	}
}

type ctxKey struct{}

// WithValues attaches structured attrs to context for automatic logging.
func WithValues(ctx context.Context, attrs ...slog.Attr) context.Context {
	if len(attrs) == 0 {
		return ctx
	}
	existing := valuesFromContext(ctx)
	merged := make([]slog.Attr, 0, len(existing)+len(attrs))
	merged = append(merged, existing...)
	merged = append(merged, attrs...)
	return context.WithValue(ctx, ctxKey{}, merged)
}

// WithValue attaches a single key/value into context for automatic logging.
func WithValue(ctx context.Context, key string, val any) context.Context {
	if key == "" {
		return ctx
	}
	return WithValues(ctx, slog.Any(key, val))
}

func valuesFromContext(ctx context.Context) []slog.Attr {
	if ctx == nil {
		return nil
	}
	if v, ok := ctx.Value(ctxKey{}).([]slog.Attr); ok {
		return v
	}
	return nil
}

type contextHandler struct {
	slog.Handler
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if values := valuesFromContext(ctx); len(values) > 0 {
		r.AddAttrs(values...)
	}
	err := h.Handler.Handle(ctx, r)
	if r.Level >= LevelFatal {
		os.Exit(1)
	}
	return err
}

type consoleHandler struct {
	w           io.Writer
	level       slog.Leveler
	addSource   bool
	replaceAttr func([]string, slog.Attr) slog.Attr
	attrs       []slog.Attr
	groups      []string
	color       bool
	sourceRoot  string
}

func (h *consoleHandler) Enabled(_ context.Context, level slog.Level) bool {
	min := slog.LevelInfo
	if h.level != nil {
		min = h.level.Level()
	}
	return level >= min
}

func (h *consoleHandler) Handle(_ context.Context, r slog.Record) error {
	if !h.Enabled(context.Background(), r.Level) {
		return nil
	}

	var buf bytes.Buffer
	ts := r.Time
	if ts.IsZero() {
		ts = time.Now()
	}
	timeStr := ts.Local().Format("2006-01-02 15:04:05.000")
	if h.color {
		timeStr = colorCyan + timeStr + colorReset
	}
	buf.WriteString(timeStr)
	buf.WriteByte(' ')

	lvl := levelText(r.Level)
	if h.color {
		lvl = colorize(r.Level, lvl)
	}
	buf.WriteByte('[')
	buf.WriteString(lvl)
	buf.WriteString("] ")

	if h.addSource {
		src := sourceFromRecord(r)
		if src.File != "" {
			path := trimSourcePath(h.sourceRoot, src.File)
			if h.color {
				path = colorBlue + path + colorReset
			}
			buf.WriteString(path)
			buf.WriteByte(':')
			lineStr := fmt.Sprintf("%d", src.Line)
			if h.color {
				lineStr = colorBlue + lineStr + colorReset
			}
			buf.WriteString(lineStr)
			buf.WriteByte(' ')
		}
	}

	msg := r.Message
	if h.color {
		msg = colorize(r.Level, msg)
	}
	buf.WriteString(msg)

	attrs := make([]slog.Attr, 0, len(h.attrs)+r.NumAttrs())
	for _, a := range h.attrs {
		attrs = appendAttr(attrs, h.groups, h.replaceAttr, a)
	}
	r.Attrs(func(a slog.Attr) bool {
		attrs = appendAttr(attrs, h.groups, h.replaceAttr, a)
		return true
	})

	for _, a := range attrs {
		key := a.Key
		val := formatValue(a.Value)
		if h.color {
			key = colorCyan + key + colorReset
			val = colorBlue + val + colorReset
		}
		buf.WriteByte(' ')
		buf.WriteString(key)
		buf.WriteByte('=')
		buf.WriteString(val)
	}

	buf.WriteByte('\n')
	_, err := h.w.Write(buf.Bytes())
	return err
}

func (h *consoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	merged := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	merged = append(merged, h.attrs...)
	merged = append(merged, attrs...)
	return &consoleHandler{
		w:           h.w,
		level:       h.level,
		addSource:   h.addSource,
		replaceAttr: h.replaceAttr,
		attrs:       merged,
		groups:      h.groups,
		color:       h.color,
		sourceRoot:  h.sourceRoot,
	}
}

func (h *consoleHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	groups := append([]string{}, h.groups...)
	groups = append(groups, name)
	return &consoleHandler{
		w:           h.w,
		level:       h.level,
		addSource:   h.addSource,
		replaceAttr: h.replaceAttr,
		attrs:       h.attrs,
		groups:      groups,
		color:       h.color,
		sourceRoot:  h.sourceRoot,
	}
}

func shouldUseColor(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	if (info.Mode() & os.ModeCharDevice) == 0 {
		return false
	}
	if term := os.Getenv("TERM"); term == "" || term == "dumb" {
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return true
}

func trimSourcePath(root, file string) string {
	if root != "" {
		if rel, err := filepath.Rel(root, file); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	return file
}

const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorMagenta = "\033[35m"
	colorYellow  = "\033[33m"
	colorGreen   = "\033[32m"
	colorBlue    = "\033[34m"
	colorCyan    = "\033[36m"
)

func colorize(level slog.Level, text string) string {
	switch {
	case level >= LevelFatal:
		return colorMagenta + text + colorReset
	case level >= slog.LevelError:
		return colorRed + text + colorReset
	case level >= slog.LevelWarn:
		return colorYellow + text + colorReset
	case level >= slog.LevelInfo:
		return colorGreen + text + colorReset
	default:
		return colorBlue + text + colorReset
	}
}

// levelText normalizes level to display text, ensuring fatal renders as FATAL instead of ERROR+N.
func levelText(level slog.Level) string {
	if level >= LevelFatal {
		return "FATAL"
	}
	return strings.ToUpper(level.String())
}

// logWithSource builds a record with caller PC to retain correct source location when called via wrappers.
// callerSkip is the additional stack frames to skip above this helper (e.g., wrapper functions).
func logWithSource(ctx context.Context, level slog.Level, msg string, args []any, callerSkip int) {
	h := slog.Default().Handler()

	pcs := make([]uintptr, 16)
	n := runtime.Callers(2+callerSkip, pcs) // skip runtime.Callers + logWithSource + wrapper(s)
	frames := runtime.CallersFrames(pcs[:n])
	var pc uintptr
	for {
		frame, more := frames.Next()
		if frame.File != "" && !strings.Contains(frame.File, "/pkg/kit/slogx/") && !strings.Contains(frame.File, "/runtime/") && !strings.Contains(frame.File, "/log/") {
			pc = frame.PC
			break
		}
		if !more {
			break
		}
	}
	rec := slog.NewRecord(time.Now(), level, msg, pc)
	rec.Add(args...)
	_ = h.Handle(ctx, rec)
	if level >= LevelFatal {
		os.Exit(1)
	}
}

func appendAttr(dst []slog.Attr, groups []string, replacer func([]string, slog.Attr) slog.Attr, a slog.Attr) []slog.Attr {
	if len(groups) > 0 {
		keyParts := append(append([]string{}, groups...), a.Key)
		a.Key = strings.Join(keyParts, ".")
	}
	if replacer != nil {
		a = replacer(groups, a)
	}
	if a.Equal(slog.Attr{}) {
		return dst
	}
	return append(dst, a)
}

func formatValue(v slog.Value) string {
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindBool:
		return strconv.FormatBool(v.Bool())
	case slog.KindInt64:
		return fmt.Sprint(v.Int64())
	case slog.KindFloat64:
		return strconv.FormatFloat(v.Float64(), 'f', -1, 64)
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindTime:
		return v.Time().Local().Format("2006-01-02 15:04:05.000")
	default:
		return fmt.Sprint(v.Any())
	}
}

func sourceFromRecord(r slog.Record) slog.Source {
	if src := r.Source(); src != nil && src.File != "" {
		return *src
	}
	if pc, file, line, ok := runtime.Caller(4); ok {
		return slog.Source{Function: runtime.FuncForPC(pc).Name(), File: file, Line: line}
	}
	return slog.Source{}
}

func defaultSourceRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

func parseLevel(v string) (slog.Level, bool) {
	switch strings.ToLower(v) {
	case "debug":
		return slog.LevelDebug, true
	case "info":
		return slog.LevelInfo, true
	case "warn", "warning":
		return slog.LevelWarn, true
	case "error", "err":
		return slog.LevelError, true
	default:
		return slog.LevelInfo, false
	}
}
