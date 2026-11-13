package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	Container          = "container"
	InitContainer      = "initContainer"
	EphemeralContainer = "ephemeralContainer"

	MaxSlugLength = 253

	slugDigestLength = 4
	slugSuffixLength = slugDigestLength*2 + 2
)

var ErrInvalidSlug = errors.New("Current inputs produce an invalid slug")

var (
	subjectWithContainer = regexp.MustCompile("apiVersion-(.+)/namespace-([^/]+)/kind-([^/]+)/name-([^/]+)/([^/]+)Name-([^/]+)")
	subjectNoContainer   = regexp.MustCompile("apiVersion-(.+)/namespace-([^/]+)/kind-([^/]+)/name-([^/]+)")
)

type Subject struct {
	APIVersion    string
	Namespace     string
	Kind          string
	Name          string
	AlternateName string
	Container     string
	ContainerKind string
	TemplateHash  string
}

func (s Subject) Validate() error {
	switch {
	case s.APIVersion == "":
		return errors.New("invalid instanceID: apiVersion cannot be empty")
	case s.Namespace == "":
		return errors.New("invalid instanceID: namespace cannot be empty")
	case s.Kind == "":
		return errors.New("invalid instanceID: kind cannot be empty")
	case s.Name == "":
		return errors.New("invalid instanceID: name cannot be empty")
	}
	return nil
}

func (s Subject) effectiveName() string {
	if s.AlternateName != "" {
		return s.AlternateName
	}
	return s.Name
}

func (s Subject) String() string {
	return s.StringNoContainer() + "/" + s.ContainerKind + "Name-" + s.Container
}

func (s Subject) StringNoContainer() string {
	return "apiVersion-" + s.APIVersion + "/namespace-" + s.Namespace + "/kind-" + s.Kind + "/name-" + s.effectiveName()
}

func (s Subject) Hashed() string {
	sum := sha256.Sum256([]byte(s.String()))
	return hex.EncodeToString(sum[:])
}

func Slug(s Subject, noContainer bool) (string, error) {
	hashed := s.Hashed()
	digests := "-" + hashed[:slugDigestLength] + "-" + hashed[len(hashed)-slugDigestLength:]
	slug := s.Kind + "-" + s.effectiveName()
	if !noContainer && s.Container != "" {
		slug += "-" + s.Container + digests
	}
	if limit := MaxSlugLength - slugSuffixLength; len(slug) >= limit {
		slug = slug[:limit] + digests
	}
	slug = strings.ToLower(slug)
	if !validSlug(slug) {
		return "", ErrInvalidSlug
	}
	return slug, nil
}

func ParseSubject(input string) (Subject, error) {
	var s Subject
	if f := subjectWithContainer.FindStringSubmatch(input); f != nil {
		s = Subject{APIVersion: f[1], Namespace: f[2], Kind: f[3], Name: f[4], ContainerKind: f[5], Container: f[6]}
	} else if f := subjectNoContainer.FindStringSubmatch(input); f != nil {
		s = Subject{APIVersion: f[1], Namespace: f[2], Kind: f[3], Name: f[4]}
	} else {
		return Subject{}, fmt.Errorf("invalid format: %s", input)
	}
	if err := s.Validate(); err != nil {
		return Subject{}, err
	}
	if s.String() != input && s.StringNoContainer() != input {
		return Subject{}, fmt.Errorf("invalid format: %s", input)
	}
	return s, nil
}

func validSlug(s string) bool {
	if len(s) < 2 || len(s) > MaxSlugLength {
		return false
	}
	if !lowerAlnum(s[0]) || !lowerAlnum(s[len(s)-1]) {
		return false
	}
	for i := 1; i < len(s)-1; i++ {
		if c := s[i]; !lowerAlnum(c) && c != '.' && c != '-' {
			return false
		}
	}
	return true
}

func lowerAlnum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}
