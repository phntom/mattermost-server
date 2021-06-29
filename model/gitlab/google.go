// Copyright (c) 2015 Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package oauthgitlab

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mattermost/mattermost-server/v5/shared/mlog"
	"hash/fnv"
	"io"
	"strings"

	"github.com/mattermost/mattermost-server/v5/einterfaces"
	"github.com/mattermost/mattermost-server/v5/model"
)

const (
	PictureURL            = "PictureURL"
	UserAuthServiceGoogle = "google"
)

type GoogleProvider struct {
}

type GoogleUser struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Locale        string `json:"locale"`
}

func init() {
	mlog.Info("Google provider is initialized")
	provider := &GoogleProvider{}
	einterfaces.RegisterOauthProvider(UserAuthServiceGoogle, provider)
}

func getRandomUsername(email string) string {
	prefix := []string{"air", "amber", "arctic", "asteroid", "astro", "bog", "bark", "blaze", "bright", "brass", "bronze", "canyon", "cavern", "celestial", "chrome", "city", "clay", "cloud", "coffee", "comet", "copper", "coral", "cosmic", "country", "cyber", "cyborg", "dark", "desert", "dew", "diamond", "dust", "dynamo", "electro", "emerald", "fire", "fire", "fire", "fire", "flare", "florescent", "fog", "forest", "frost", "galactic", "giant", "glacier", "glass", "gold", "grass", "gust", "helium", "hydro", "hyper", "hypno", "ice", "icicle", "iron", "jungle", "krypton", "lagoon", "lake", "lava", "light", "lightning", "lunar", "macro", "magic", "magma", "magnetic", "matrix", "mecha", "mega", "metal", "meteor", "mini", "mist", "monsoon", "moon", "mountain", "neon", "nickel", "nimbus", "nitro", "nyan", "ocean", "oxygen", "ozone", "polar", "pollen", "pond", "power", "puddle", "quake", "radioactive", "rain", "river", "robo", "rock", "ruby", "sand", "sapphire", "sea", "silicon", "silver", "sky", "sleet", "smoke", "solar", "solstice", "space", "spark", "spectro", "star", "steam", "steel", "stellar", "stone", "storm", "sun", "super", "supercalifragilisticexpialidocious", "supernova", "surf", "swamp", "techno", "thunder", "tidal", "tin", "tree", "tropic", "tundra", "turbo", "urban", "valley", "vapor", "volcano", "vortex", "water", "wave", "web", "wind", "yttrium", "zinc"}
	suffix := []string{"aardvark", "adder", "alligator", "amoeba", "anemone", "ant", "antelope", "antlion", "apatosaurus", "ape", "arachnid", "baboon", "badger", "bass", "bat", "bear", "beast", "beaver", "bee", "beetle", "bird", "bobcat", "brontosaurus", "buffalo", "bug", "bumblebee", "bunny", "butterfly", "camel", "canary", "canine", "carp", "cat", "chameleon", "cheetah", "chicken", "chipmunk", "cicada", "clam", "condor", "cougar", "cow", "coyote", "crab", "crane", "cricket", "crocodile", "crow", "curlew", "deer", "dingo", "dinosaur", "dodo", "dog", "dolphin", "donkey", "dragon", "droid", "duck", "eagle", "elephant", "elk", "emu", "feline", "ferret", "finch", "fish", "flea", "fox", "frog", "fly", "gazelle", "gecko", "giraffe", "gnat", "gnome", "goat", "goose", "gopher", "gorilla", "grub", "hamster", "hare", "hawk", "hippopotamus", "hog", "horse", "husky", "hyena", "iguana", "impala", "jackal", "jackalope", "jaguar", "jay", "jellyfish", "kangaroo", "kitten", "klumpen", "koala", "koi", "krill", "leech", "lemming", "lemur", "leopard", "lion", "lizard", "llama", "lobster", "loon", "lynx", "mantis", "marmot", "martian", "mole", "mollusk", "mongoose", "monkey", "moose", "mosquito", "moth", "mouse", "mule", "narwhal", "newt", "octopus", "orca", "oriole", "ostrich", "otter", "owl", "oyster", "panda", "panther", "parakeet", "parrot", "pelican", "penguin", "phoenix", "pig", "pigeon", "phoenix", "piranha", "platypus", "pony", "porcupine", "possum", "primate", "pterodactyl", "puck", "python", "quail", "rabbit", "raccoon", "raptor", "rat", "reindeer", "rhino", "roadrunner", "robin", "robot", "salamander", "salmon", "scorpion", "seal", "shark", "sheep", "shrimp", "skunk", "sloth", "slug", "snake", "sparrow", "sphinx", "spider", "squid", "squirrel", "stag", "stork", "swan", "tarantula", "tiger", "toad", "tortoise", "triceratops", "trout", "tuna", "turkey", "turtle", "unicorn", "viper", "vulture", "walrus", "wasp", "weasel", "whale", "wolf", "worm", "wren", "yak", "zebra", "zebu"}
	h := fnv.New32a()
	h.Write([]byte(email))
	hash := int(h.Sum32())

	prefixIndex := (hash / 1000) % len(prefix)
	suffixIndex := (hash) % len(suffix)

	return fmt.Sprintf("%s%s", prefix[prefixIndex], suffix[suffixIndex])
}

func userFromGoogleUser(gou *GoogleUser) *model.User {
	user := &model.User{}
	user.Username = getRandomUsername(gou.Email)
	user.FirstName = gou.GivenName
	user.LastName = gou.FamilyName
	user.Email = strings.ToLower(gou.Email)
	user.EmailVerified = gou.EmailVerified
	if model.IsValidLocale(gou.Locale) {
		user.Locale = gou.Locale
	}
	if len(gou.Picture) > 0 {
		user.SetProp(PictureURL, gou.Picture)
	}
	userId := gou.getAuthData()
	user.AuthData = &userId
	user.AuthService = UserAuthServiceGoogle

	return user
}

func googleUserFromJson(data io.Reader) (*GoogleUser, error) {
	decoder := json.NewDecoder(data)
	var gou GoogleUser
	err := decoder.Decode(&gou)
	if err != nil {
		return nil, err
	}
	return &gou, nil
}

func (gou *GoogleUser) ToJson() string {
	b, err := json.Marshal(gou)
	if err != nil {
		return ""
	}
	return string(b)
}

func (gou *GoogleUser) IsValid() error {
	if len(gou.Sub) == 0 {
		return errors.New("user sub can't be empty")
	}

	if gou.Email == "" {
		return errors.New("user e-mail should not be empty")
	}

	if gou.EmailVerified == false {
		return errors.New("email is not verified")
	}

	return nil
}

func (gou *GoogleUser) getAuthData() string {
	return gou.Sub
}

func (m *GoogleProvider) GetUserFromJson(data io.Reader, tokenUser *model.User) (*model.User, error) {
	gou, err := googleUserFromJson(data)
	if err != nil {
		return nil, err
	}
	if err = gou.IsValid(); err != nil {
		return nil, err
	}

	return userFromGoogleUser(gou), nil
}

func (m *GoogleProvider) GetSSOSettings(config *model.Config, service string) (*model.SSOSettings, error) {
	return &config.GoogleSettings, nil
}

func (m *GoogleProvider) GetUserFromIdToken(idToken string) (*model.User, error) {
	return nil, nil
}

func (m *GoogleProvider) IsSameUser(dbUser, oauthUser *model.User) bool {
	return dbUser.AuthService == oauthUser.AuthService &&
		dbUser.AuthData == oauthUser.AuthData
}
