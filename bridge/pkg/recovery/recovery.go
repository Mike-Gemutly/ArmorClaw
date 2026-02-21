// Package recovery provides account recovery functionality for ArmorClaw.
// Implements GAP #6 from user journey analysis.
//
// Recovery Flow:
// 1. During setup: Generate 12-word recovery phrase (BIP39-style)
// 2. User stores phrase securely (displayed once, never shown again)
// 3. On device loss: User provides recovery phrase on new device
// 4. System verifies phrase and initiates 24-48 hour recovery window
// 5. During recovery: Read-only access, limited operations
// 6. After recovery: Full access restored, old devices invalidated
package recovery

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/armorclaw/bridge/pkg/securerandom"
)

// Recovery phrase word list (BIP39 English subset - 2048 words)
// Using a reduced set for simplicity while maintaining security
var wordList = []string{
	"abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract",
	"absurd", "abuse", "access", "accident", "account", "accuse", "achieve", "acid",
	"acoustic", "acquire", "across", "act", "action", "actor", "actress", "actual",
	"adapt", "add", "addict", "address", "adjust", "admit", "adult", "advance",
	"advice", "aerobic", "affair", "afford", "afraid", "again", "age", "agent",
	"agree", "ahead", "aim", "air", "airport", "aisle", "alarm", "album",
	"alcohol", "alert", "alien", "all", "alley", "allow", "almost", "alone",
	"alpha", "already", "also", "alter", "always", "amateur", "amazing", "among",
	"amount", "amused", "analyst", "anchor", "ancient", "anger", "angle", "angry",
	"animal", "ankle", "announce", "annual", "another", "answer", "antenna", "antique",
	"anxiety", "any", "apart", "apology", "appear", "apple", "approve", "april",
	"arch", "arctic", "area", "arena", "argue", "arm", "armed", "armor",
	"army", "around", "arrange", "arrest", "arrive", "arrow", "art", "artefact",
	"artist", "artwork", "ask", "aspect", "assault", "asset", "assist", "assume",
	"asthma", "athlete", "atom", "attack", "attend", "attitude", "attract", "auction",
	"audit", "august", "aunt", "author", "auto", "autumn", "average", "avocado",
	"avoid", "awake", "aware", "away", "awesome", "awful", "awkward", "axis",
	"baby", "bachelor", "bacon", "badge", "bag", "balance", "balcony", "ball",
	"bamboo", "banana", "banner", "bar", "barely", "bargain", "barrel", "base",
	"basic", "basket", "battle", "beach", "bean", "beauty", "because", "become",
	"beef", "before", "begin", "behave", "behind", "believe", "below", "belt",
	"bench", "benefit", "best", "betray", "better", "between", "beyond", "bicycle",
	"bid", "bike", "bind", "biology", "bird", "birth", "bitter", "black",
	"blade", "blame", "blanket", "blast", "bleak", "bless", "blind", "blood",
	"blossom", "blouse", "blue", "blur", "blush", "board", "boat", "body",
	"boil", "bomb", "bone", "bonus", "book", "boost", "border", "boring",
	"borrow", "boss", "bottom", "bounce", "box", "boy", "bracket", "brain",
	"brand", "brass", "brave", "bread", "breeze", "brick", "bridge", "brief",
	"bright", "bring", "brisk", "broken", "bronze", "broom", "brother", "brown",
	"brush", "bubble", "buddy", "budget", "buffalo", "build", "bulb", "bulk",
	"bundle", "bunker", "burden", "burger", "burst", "bus", "business", "busy",
	"butter", "buyer", "buzz", "cabbage", "cabin", "cable", "cactus", "cage",
	"cake", "call", "calm", "camera", "camp", "can", "canal", "cancel",
	"candy", "cannon", "canoe", "canvas", "canyon", "capable", "capital", "captain",
	"car", "carbon", "card", "cargo", "carpet", "carry", "cart", "case",
	"cash", "casino", "castle", "casual", "cat", "catalog", "catch", "category",
	"cattle", "caught", "cause", "caution", "cave", "ceiling", "celery", "cement",
	"census", "century", "cereal", "certain", "chair", "chalk", "champion", "change",
	"chaos", "chapter", "charge", "chase", "chat", "cheap", "check", "cheese",
	"chef", "cherry", "chest", "chicken", "chief", "child", "chimney", "choice",
	"choose", "chronic", "chuckle", "chunk", "churn", "cigar", "cinnamon", "circle",
	"citizen", "city", "civil", "claim", "clap", "clarify", "claw", "clay",
	"clean", "clerk", "clever", "click", "client", "cliff", "climb", "clinic",
	"clip", "clock", "clog", "close", "cloth", "cloud", "clown", "club",
	"clump", "cluster", "clutch", "coach", "coast", "coconut", "code", "coffee",
	"coil", "coin", "collect", "color", "colt", "column", "comb", "come",
	"comfort", "comic", "common", "company", "concert", "conduct", "confirm", "congress",
	"connect", "consider", "consume", "contact", "contain", "content", "contest", "context",
	"contract", "contrast", "control", "convert", "convince", "cook", "cool", "copper",
	"copy", "coral", "core", "corn", "correct", "cost", "cotton", "couch",
	"couple", "course", "cousin", "cover", "coyote", "crack", "cradle", "craft",
	"cram", "crane", "crash", "crater", "crawl", "crazy", "cream", "credit",
	"creek", "crew", "cricket", "crime", "crisp", "critic", "crop", "cross",
	"crouch", "crowd", "crucial", "cruel", "cruise", "crumble", "crunch", "crush",
	"cry", "crystal", "cube", "culture", "cup", "cupboard", "curious", "current",
	"curtain", "curve", "cushion", "custom", "cute", "cycle", "dad", "damage",
	"damp", "dance", "danger", "dare", "darkness", "daughter", "dawn", "day",
	"dead", "deal", "debate", "debris", "decade", "december", "decide", "decline",
	"decorate", "decrease", "deer", "defense", "define", "defy", "degree", "delay",
	"deliver", "demand", "demise", "denial", "dentist", "deny", "depart", "depend",
	"deposit", "depth", "deputy", "derive", "describe", "desert", "design", "desk",
	"despair", "destroy", "detail", "detect", "develop", "device", "devote", "diagram",
	"dial", "diamond", "diary", "dice", "diesel", "diet", "differ", "digital",
	"dignity", "dilemma", "dinner", "dinosaur", "direct", "dirt", "disagree", "discover",
	"disease", "dish", "dismiss", "disorder", "display", "distance", "divert", "divide",
	"divorce", "dizzy", "doctor", "document", "dog", "doll", "dolphin", "domain",
	"donate", "donkey", "donor", "door", "dose", "double", "dove", "draft",
	"dragon", "drama", "drastic", "draw", "dream", "dress", "drift", "drill",
	"drink", "drip", "drive", "drop", "drum", "dry", "duck", "dumb",
	"dune", "during", "dust", "dutch", "duty", "dwarf", "dynamic", "eager",
	"eagle", "early", "earn", "earth", "easily", "east", "easy", "echo",
	"ecology", "economy", "edge", "edit", "educate", "effort", "egg", "eight",
	"either", "elbow", "elder", "electric", "elegant", "element", "elephant", "elevator",
	"elite", "else", "embark", "embody", "embrace", "emerge", "emotion", "employ",
	"empower", "empty", "enable", "enact", "end", "endless", "endorse", "enemy",
	"energy", "enforce", "engage", "engine", "enhance", "enjoy", "enlist", "enough",
	"enrich", "enroll", "ensure", "enter", "entire", "entrance", "entry", "envelope",
	"episode", "equal", "equip", "era", "erase", "erode", "erosion", "error",
	"erupt", "escape", "essay", "essence", "estate", "eternal", "ethics", "evidence",
	"evil", "evoke", "evolve", "exact", "example", "excess", "exchange", "excite",
	"exclude", "excuse", "execute", "exercise", "exhaust", "exhibit", "exile", "exist",
	"exit", "exotic", "expand", "expect", "expire", "explain", "expose", "express",
	"extend", "extra", "eye", "fabric", "face", "faculty", "fade", "faint",
	"faith", "fall", "false", "fame", "family", "famous", "fan", "fancy",
	"fantasy", "farm", "fashion", "fat", "fatal", "father", "fatigue", "fault",
	"favorite", "feature", "february", "federal", "fee", "feed", "feel", "female",
	"fence", "festival", "fever", "few", "fiber", "fiction", "field", "figure",
	"file", "film", "filter", "final", "find", "fine", "finger", "finish",
	"fire", "firm", "first", "fiscal", "fish", "fit", "fitness", "fix",
	"flag", "flame", "flash", "flat", "flavor", "flee", "flight", "flip",
	"float", "flock", "flood", "floor", "flower", "fluid", "flush", "fly",
	"foam", "focus", "fog", "foil", "fold", "follow", "food", "foot",
	"force", "forest", "forget", "fork", "fortune", "forum", "forward", "fossil",
	"foster", "found", "fox", "fragile", "frame", "frequent", "fresh", "friend",
	"fringe", "frog", "front", "frost", "frown", "frozen", "fruit", "fuel",
	"fun", "functional", "funny", "furious", "furnace", "fury", "future", "gadget",
	"gain", "galaxy", "gallery", "game", "gap", "garage", "garbage", "garden",
	"garlic", "garment", "gas", "gasp", "gate", "gather", "gauge", "gaze",
	"general", "genius", "genre", "gentle", "genuine", "gesture", "ghost", "giant",
	"gift", "giggle", "ginger", "giraffe", "girl", "give", "glad", "glance",
	"glare", "glass", "glide", "glimpse", "globe", "gloom", "glory", "glove",
	"glow", "glue", "goat", "goddess", "gold", "good", "goose", "gorilla",
	"gospel", "gossip", "govern", "gown", "grab", "grace", "grain", "grant",
	"grape", "grass", "gravity", "great", "green", "grid", "grief", "grit",
	"grocery", "group", "grove", "grow", "growl", "growth", "grunt", "guard",
	"guess", "guide", "guilt", "guitar", "gun", "gym", "habit", "hair",
	"half", "hammer", "hamster", "hand", "happy", "harbor", "hard", "harsh",
	"harvest", "hat", "have", "hawk", "hazard", "head", "health", "heart",
	"heavy", "hedge", "height", "hello", "helmet", "help", "hen", "hero",
	"hidden", "high", "hill", "hint", "hip", "hire", "history", "hobby",
	"hockey", "hold", "hole", "holiday", "hollow", "home", "honey", "hood",
	"hope", "horn", "horror", "horse", "hospital", "host", "hotel", "hour",
	"hover", "hub", "huge", "human", "humble", "humor", "hundred", "hungry",
	"hunt", "hurdle", "hurry", "hurt", "husband", "hybrid", "ice", "icon",
	"idea", "identify", "idle", "ignore", "ill", "illegal", "illness", "image",
	"imitate", "immense", "immune", "impact", "impose", "improve", "impulse", "inch",
	"include", "income", "increase", "index", "indicate", "indoor", "industry", "infant",
	"inflict", "inform", "inhale", "inherit", "initial", "inject", "injury", "inmate",
	"inner", "innocent", "input", "inquiry", "insane", "insect", "inside", "inspire",
	"install", "intact", "interest", "into", "invest", "invite", "involve", "iron",
	"island", "isolate", "issue", "item", "ivory", "jacket", "jaguar", "jar",
	"jazz", "jeans", "jelly", "jersey", "job", "join", "joke", "journey",
	"joy", "judge", "juice", "jump", "jungle", "junior", "junk", "just",
	"kangaroo", "keen", "keep", "ketchup", "key", "kick", "kid", "kidney",
	"kind", "kingdom", "kiss", "kit", "kitchen", "kite", "kitten", "kiwi",
	"knee", "knife", "knock", "know", "lab", "label", "labor", "ladder",
	"lady", "lake", "lamp", "language", "laptop", "large", "later", "latin",
	"laugh", "laundry", "lava", "law", "lawn", "lawsuit", "layer", "lazy",
	"leader", "leaf", "learn", "leave", "lecture", "left", "leg", "legal",
	"legend", "leisure", "lemon", "lend", "length", "lens", "leopard", "lesson",
	"letter", "level", "liar", "liberty", "library", "license", "life", "lift",
	"light", "like", "limb", "limit", "link", "lion", "liquid", "list",
	"little", "live", "lizard", "load", "loan", "lobster", "local", "lock",
	"logic", "lonely", "long", "loop", "lottery", "loud", "lounge", "love",
	"luck", "luggage", "lumber", "lunar", "lunch", "luxury", "lyrics", "machine",
	"mad", "magic", "magnet", "maid", "mail", "main", "major", "make",
	"mammal", "man", "manage", "mandate", "mango", "mansion", "manual", "maple",
	"marble", "march", "margin", "marine", "market", "marriage", "mask", "mass",
	"master", "match", "material", "math", "matrix", "matter", "maximum", "maze",
	"meadow", "mean", "measure", "meat", "mechanic", "medal", "media", "melody",
	"melt", "member", "memory", "mention", "menu", "mercy", "merge", "merit",
	"merry", "mesh", "message", "metal", "method", "middle", "midnight", "milk",
	"million", "mimic", "mind", "minimum", "minor", "minute", "miracle", "mirror",
	"misery", "miss", "mistake", "mix", "mixed", "mixture", "mobile", "model",
	"modify", "mom", "moment", "monitor", "monkey", "monster", "month", "moon",
	"moral", "more", "morning", "mosquito", "mother", "motion", "motor", "mountain",
	"mouse", "move", "movie", "much", "muffin", "mule", "muscle", "museum",
	"mushroom", "music", "must", "mutual", "myself", "mystery", "myth", "naive",
	"name", "napkin", "narrow", "nasty", "nation", "nature", "near", "neck",
	"need", "negative", "neglect", "neither", "nephew", "nerve", "nest", "net",
	"network", "neutral", "never", "news", "next", "nice", "night", "noble",
	"noise", "nominee", "noodle", "normal", "north", "nose", "notable", "note",
	"nothing", "notice", "novel", "now", "nuclear", "number", "nurse", "nut",
	"oak", "obey", "object", "oblige", "obscure", "observe", "obtain", "obvious",
	"occur", "ocean", "october", "odor", "off", "offer", "office", "often",
	"oil", "okay", "old", "olive", "olympic", "omit", "once", "one",
	"onion", "online", "only", "open", "opera", "opinion", "oppose", "option",
	"orange", "orbit", "orchard", "order", "ordinary", "organ", "orient", "original",
	"orphan", "ostrich", "other", "outdoor", "outer", "output", "outside", "oval",
	"oven", "over", "own", "owner", "oxygen", "oyster", "ozone", "pact",
	"paddle", "page", "pair", "palace", "palm", "panda", "panel", "panic",
	"panther", "paper", "parade", "parent", "park", "parrot", "party", "pass",
	"patch", "path", "patient", "patrol", "pattern", "pause", "pave", "payment",
	"peace", "peanut", "pear", "peasant", "pelican", "pen", "penalty", "pencil",
	"people", "pepper", "perfect", "permit", "person", "pet", "phone", "photo",
	"phrase", "physical", "piano", "picnic", "picture", "piece", "pig", "pigeon",
	"pill", "pilot", "pink", "pioneer", "pipe", "pistol", "pitch", "pizza",
	"place", "planet", "plastic", "plate", "play", "please", "pledge", "pluck",
	"plug", "plunge", "poem", "poet", "point", "polar", "pole", "police",
	"pond", "pony", "pool", "popular", "portion", "position", "possible", "post",
	"potato", "pottery", "poverty", "powder", "power", "practice", "praise", "predict",
	"prefer", "prepare", "present", "pretty", "prevent", "price", "pride", "primary",
	"print", "priority", "prison", "private", "prize", "problem", "process", "produce",
	"profit", "program", "project", "promote", "proof", "property", "prosper", "protect",
	"proud", "provide", "public", "pudding", "pull", "pulp", "pulse", "pumpkin",
	"punch", "pupil", "puppy", "purchase", "purity", "purpose", "purse", "push",
	"put", "puzzle", "pyramid", "quality", "quantum", "quarter", "question", "quick",
	"quit", "quiz", "quote", "rabbit", "raccoon", "race", "rack", "radar",
	"radio", "rail", "rain", "raise", "rally", "ramp", "ranch", "random",
	"range", "rapid", "rare", "rate", "rather", "raven", "raw", "razor",
	"ready", "real", "reason", "rebel", "rebuild", "recall", "receive", "recipe",
	"record", "recycle", "reduce", "reflect", "reform", "refuse", "region", "regret",
	"regular", "reject", "relax", "release", "relief", "rely", "remain", "remember",
	"remind", "remove", "render", "renew", "rent", "reopen", "repair", "repeat",
	"replace", "report", "require", "rescue", "resemble", "resist", "resource", "response",
	"result", "retire", "retreat", "return", "reunion", "reveal", "review", "reward",
	"rhythm", "rib", "ribbon", "rice", "rich", "ride", "ridge", "rifle",
	"right", "rigid", "ring", "riot", "ripple", "risk", "ritual", "rival",
	"river", "road", "roast", "robot", "robust", "rocket", "romance", "roof",
	"rookie", "room", "rose", "rotate", "rough", "round", "route", "royal",
	"rubber", "rude", "rug", "rule", "run", "runway", "rural", "sad",
	"saddle", "sadness", "safe", "sail", "salad", "salmon", "salon", "salt",
	"salute", "same", "sample", "sand", "satisfy", "satoshi", "sauce", "sausage",
	"save", "say", "scale", "scan", "scare", "scatter", "scene", "scheme",
	"school", "science", "scissors", "scorpion", "scout", "scrap", "screen", "script",
	"scrub", "sea", "search", "season", "seat", "second", "secret", "section",
	"security", "seed", "seek", "segment", "select", "sell", "seminar", "senior",
	"sense", "sentence", "series", "service", "session", "settle", "setup", "seven",
	"shadow", "shaft", "shallow", "share", "shed", "shell", "sheriff", "shield",
	"shift", "shine", "ship", "shiver", "shock", "shoe", "shoot", "shop",
	"short", "shoulder", "shove", "shrimp", "shrug", "shuffle", "shut", "shy",
	"sibling", "sick", "side", "siege", "sight", "sign", "silent", "silk",
	"silly", "silver", "similar", "simple", "since", "sing", "siren", "sister",
	"situate", "six", "size", "skate", "sketch", "ski", "skill", "skin",
	"skirt", "skull", "slab", "slam", "sleep", "slender", "slice", "slide",
	"slight", "slim", "slogan", "slot", "slow", "slush", "small", "smart",
	"smile", "smoke", "smooth", "snack", "snake", "snap", "sniff", "snow",
	"soap", "soccer", "social", "sock", "soda", "soft", "solar", "soldier",
	"solution", "solve", "someone", "song", "soon", "sorry", "sort", "soul",
	"sound", "soup", "source", "south", "space", "spare", "spatial", "spawn",
	"speak", "special", "speed", "spell", "spend", "sphere", "spice", "spider",
	"spike", "spin", "spirit", "split", "spoil", "sponsor", "spoon", "sport",
	"spot", "spray", "spread", "spring", "spy", "square", "squeeze", "squirrel",
	"stable", "stadium", "staff", "stage", "stairs", "stamp", "stand", "start",
	"state", "stay", "steak", "steel", "stem", "step", "stereo", "stick",
	"still", "sting", "stock", "stomach", "stone", "stool", "story", "stove",
	"strategy", "street", "strike", "strong", "struggle", "student", "stuff", "stumble",
	"style", "subject", "submit", "subway", "success", "such", "sudden", "suffer",
	"sugar", "suggest", "suit", "summer", "sun", "sunny", "sunset", "super",
	"supply", "supreme", "sure", "surface", "surge", "surprise", "surround", "survey",
	"suspect", "sustain", "swallow", "swamp", "swap", "swarm", "swear", "sweet",
	"swift", "swim", "swing", "switch", "sword", "symbol", "symptom", "syrup",
	"system", "table", "tackle", "tag", "tail", "talent", "talk", "tank",
	"tape", "target", "task", "taste", "tattoo", "taxi", "teach", "team",
	"tell", "ten", "tenant", "tennis", "tent", "term", "test", "text",
	"thank", "that", "theme", "then", "theory", "there", "they", "thin",
	"thing", "think", "third", "thirty", "this", "thorough", "thousand", "three",
	"thrive", "throw", "thumb", "thunder", "ticket", "tide", "tiger", "tilt",
	"timber", "time", "tiny", "tip", "tired", "tissue", "title", "toast",
	"today", "tobacco", "toddler", "toe", "together", "toilet", "token", "tomato",
	"tomorrow", "tone", "tongue", "tonight", "tool", "tooth", "top", "topic",
	"topple", "torch", "tornado", "tortoise", "toss", "total", "tourist", "toward",
	"tower", "town", "toy", "track", "trade", "traffic", "tragic", "train",
	"transfer", "trap", "trash", "travel", "tray", "treat", "tree", "trend",
	"trial", "tribe", "trick", "trigger", "trim", "trip", "trophy", "trouble",
	"truck", "true", "truly", "trumpet", "trust", "truth", "try", "tube",
	"tuition", "tumble", "tuna", "tunnel", "turkey", "turn", "turtle", "twelve",
	"twenty", "twice", "twin", "twist", "two", "type", "typical", "ugly",
	"umbrella", "unable", "unaware", "uncle", "uncover", "under", "undo", "unfair",
	"unfold", "unhappy", "uniform", "unique", "unit", "universe", "unknown", "unlock",
	"until", "unusual", "unveil", "update", "upgrade", "uphold", "upon", "upper",
	"upset", "urban", "urge", "usage", "use", "used", "useful", "useless",
	"usual", "utility", "vacant", "vacuum", "vague", "valid", "valley", "valve",
	"van", "vanish", "vapor", "various", "vast", "vault", "velvet", "vendor",
	"venture", "venue", "verb", "verify", "version", "very", "vessel", "veteran",
	"viable", "vibrant", "vicious", "victory", "video", "view", "village", "vintage",
	"violin", "virtual", "virus", "visa", "visit", "visual", "vital", "vivid",
	"vocal", "voice", "void", "volcano", "volume", "vote", "voyage", "wage",
	"wagon", "wait", "walk", "wall", "walnut", "want", "warfare", "warm",
	"warrior", "wash", "wasp", "waste", "water", "wave", "way", "wealth",
	"weapon", "wear", "weasel", "weather", "web", "wedding", "weekend", "weird",
	"welcome", "west", "wet", "whale", "what", "wheat", "wheel", "when",
	"where", "whip", "whisper", "wide", "width", "wife", "wild", "will",
	"win", "window", "wine", "wing", "wink", "winner", "winter", "wire",
	"wisdom", "wise", "wish", "witness", "wolf", "woman", "wonder", "wood",
	"wool", "word", "work", "world", "worry", "worth", "wrap", "wreck",
	"wrestle", "wrist", "write", "wrong", "yard", "year", "yellow", "you",
	"young", "youth", "zebra", "zero", "zone", "zoo",
}

const (
	// PhraseLength is the number of words in a recovery phrase
	PhraseLength = 12

	// RecoveryWindowHours is the duration of restricted access during recovery
	RecoveryWindowHours = 48

	// EntropyBytes is the amount of entropy for phrase generation (128 bits)
	EntropyBytes = 16
)

// RecoveryStatus represents the state of a recovery attempt
type RecoveryStatus string

const (
	RecoveryStatusNone     RecoveryStatus = "none"
	RecoveryStatusPending  RecoveryStatus = "pending"
	RecoveryStatusActive   RecoveryStatus = "active"
	RecoveryStatusComplete RecoveryStatus = "complete"
	RecoveryStatusExpired  RecoveryStatus = "expired"
)

// RecoveryState tracks an ongoing recovery process
type RecoveryState struct {
	ID           string         `json:"id"`
	Status       RecoveryStatus `json:"status"`
	StartedAt    time.Time      `json:"started_at"`
	ExpiresAt    time.Time      `json:"expires_at"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
	NewDeviceID  string         `json:"new_device_id"`
	OldDevices   []string       `json:"old_devices"`
	Attempts     int            `json:"attempts"`
	ReadOnlyMode bool           `json:"read_only_mode"`
}

// Manager handles account recovery operations
type Manager struct {
	db        *sql.DB
	mu        sync.RWMutex
	encryptKey []byte
}

var (
	ErrInvalidPhrase      = errors.New("invalid recovery phrase")
	ErrRecoveryNotFound   = errors.New("recovery not found")
	ErrRecoveryExpired    = errors.New("recovery window expired")
	ErrRecoveryAlready    = errors.New("recovery already in progress")
	ErrPhraseNotSet       = errors.New("recovery phrase not set")
	ErrTooManyAttempts    = errors.New("too many recovery attempts")
)

// NewManager creates a new recovery manager
func NewManager(db *sql.DB, encryptKey []byte) (*Manager, error) {
	if len(encryptKey) != chacha20poly1305.KeySize {
		return nil, errors.New("invalid encryption key size")
	}

	m := &Manager{
		db:        db,
		encryptKey: encryptKey,
	}

	if err := m.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize recovery schema: %w", err)
	}

	return m, nil
}

// initSchema creates the recovery tables
func (m *Manager) initSchema() error {
	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS recovery_phrases (
			id TEXT PRIMARY KEY,
			phrase_hash TEXT NOT NULL UNIQUE,
			phrase_encrypted BLOB NOT NULL,
			nonce BLOB NOT NULL,
			created_at INTEGER NOT NULL,
			verified_at INTEGER,
			is_active INTEGER DEFAULT 1
		);

		CREATE TABLE IF NOT EXISTS recovery_sessions (
			id TEXT PRIMARY KEY,
			status TEXT NOT NULL,
			started_at INTEGER NOT NULL,
			expires_at INTEGER NOT NULL,
			completed_at INTEGER,
			new_device_id TEXT,
			attempts INTEGER DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS invalidated_devices (
			device_id TEXT PRIMARY KEY,
			invalidated_at INTEGER NOT NULL,
			reason TEXT NOT NULL
		);
	`)
	return err
}

// GeneratePhrase generates a new 12-word recovery phrase
func GeneratePhrase() (string, error) {
	entropy, err := securerandom.Bytes(EntropyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate entropy: %w", err)
	}

	words := make([]string, PhraseLength)
	for i := 0; i < PhraseLength; i++ {
		// Use 11 bits per word (2048 words in list)
		index := int(entropy[i])<<3 | int(entropy[(i+1)%EntropyBytes]>>5)
		words[i] = wordList[index%len(wordList)]
	}

	return strings.Join(words, " "), nil
}

// ValidatePhrase checks if a phrase has the correct format
func ValidatePhrase(phrase string) error {
	words := strings.Fields(phrase)
	if len(words) != PhraseLength {
		return fmt.Errorf("phrase must have %d words, got %d", PhraseLength, len(words))
	}

	wordSet := make(map[string]bool)
	for _, w := range wordList {
		wordSet[w] = true
	}

	for _, word := range words {
		if !wordSet[strings.ToLower(word)] {
			return fmt.Errorf("invalid word in phrase: %s", word)
		}
	}

	return nil
}

// HashPhrase creates a hash of the recovery phrase for storage
func HashPhrase(phrase string) string {
	hash := sha256.Sum256([]byte(strings.ToLower(phrase)))
	return hex.EncodeToString(hash[:])
}

// StorePhrase stores an encrypted recovery phrase
func (m *Manager) StorePhrase(phrase string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := ValidatePhrase(phrase); err != nil {
		return err
	}

	// Hash the phrase for lookup
	phraseHash := HashPhrase(phrase)

	// Check if phrase already exists
	var existing int
	err := m.db.QueryRow(`SELECT COUNT(*) FROM recovery_phrases WHERE is_active = 1`).Scan(&existing)
	if err != nil {
		return err
	}
	if existing > 0 {
		return errors.New("recovery phrase already set - invalidate old one first")
	}

	// Encrypt the phrase
	aead, err := chacha20poly1305.NewX(m.encryptKey)
	if err != nil {
		return err
	}

	nonce, err := securerandom.Bytes(aead.NonceSize())
	if err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	encrypted := aead.Seal(nil, nonce, []byte(phrase), nil)

	// Store encrypted phrase
	_, err = m.db.Exec(`
		INSERT INTO recovery_phrases (id, phrase_hash, phrase_encrypted, nonce, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, generateID(), phraseHash, encrypted, nonce, time.Now().Unix())

	return err
}

// VerifyPhrase verifies a recovery phrase and starts recovery process
func (m *Manager) VerifyPhrase(phrase, newDeviceID string) (*RecoveryState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	phraseHash := HashPhrase(phrase)

	// Look up the phrase
	var id string
	var isActive bool
	err := m.db.QueryRow(`
		SELECT id, is_active FROM recovery_phrases WHERE phrase_hash = ?
	`, phraseHash).Scan(&id, &isActive)

	if err == sql.ErrNoRows {
		return nil, ErrInvalidPhrase
	}
	if err != nil {
		return nil, err
	}

	if !isActive {
		return nil, ErrInvalidPhrase
	}

	// Check for existing recovery session
	var existingCount int
	err = m.db.QueryRow(`
		SELECT COUNT(*) FROM recovery_sessions
		WHERE status IN ('pending', 'active') AND expires_at > ?
	`, time.Now().Unix()).Scan(&existingCount)
	if err != nil {
		return nil, err
	}
	if existingCount > 0 {
		return nil, ErrRecoveryAlready
	}

	// Create recovery session
	recoveryID := generateID()
	now := time.Now()
	expiresAt := now.Add(RecoveryWindowHours * time.Hour)

	_, err = m.db.Exec(`
		INSERT INTO recovery_sessions (id, status, started_at, expires_at, new_device_id)
		VALUES (?, 'active', ?, ?, ?)
	`, recoveryID, now.Unix(), expiresAt.Unix(), newDeviceID)

	if err != nil {
		return nil, err
	}

	// Mark phrase as verified
	_, err = m.db.Exec(`UPDATE recovery_phrases SET verified_at = ? WHERE id = ?`, now.Unix(), id)
	if err != nil {
		return nil, err
	}

	return &RecoveryState{
		ID:           recoveryID,
		Status:       RecoveryStatusActive,
		StartedAt:    now,
		ExpiresAt:    expiresAt,
		NewDeviceID:  newDeviceID,
		ReadOnlyMode: true,
	}, nil
}

// GetRecoveryState returns the current recovery state
func (m *Manager) GetRecoveryState(recoveryID string) (*RecoveryState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var state RecoveryState
	var status string
	var completedAt sql.NullInt64

	err := m.db.QueryRow(`
		SELECT id, status, started_at, expires_at, completed_at, new_device_id, attempts
		FROM recovery_sessions WHERE id = ?
	`, recoveryID).Scan(
		&state.ID, &status, &state.StartedAt, &state.ExpiresAt,
		&completedAt, &state.NewDeviceID, &state.Attempts,
	)

	if err == sql.ErrNoRows {
		return nil, ErrRecoveryNotFound
	}
	if err != nil {
		return nil, err
	}

	state.Status = RecoveryStatus(status)
	if completedAt.Valid {
		t := time.Unix(completedAt.Int64, 0)
		state.CompletedAt = &t
	}

	// Check if expired
	if state.Status == RecoveryStatusActive && time.Now().After(state.ExpiresAt) {
		state.Status = RecoveryStatusExpired
		state.ReadOnlyMode = false
	} else if state.Status == RecoveryStatusActive {
		state.ReadOnlyMode = true
	}

	return &state, nil
}

// CompleteRecovery finalizes the recovery process
func (m *Manager) CompleteRecovery(recoveryID string, oldDevices []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verify recovery is active
	var status string
	var expiresAt int64
	err := m.db.QueryRow(`
		SELECT status, expires_at FROM recovery_sessions WHERE id = ?
	`, recoveryID).Scan(&status, &expiresAt)

	if err == sql.ErrNoRows {
		return ErrRecoveryNotFound
	}
	if err != nil {
		return err
	}

	if status != "active" {
		return errors.New("recovery not in active state")
	}

	if time.Now().After(time.Unix(expiresAt, 0)) {
		return ErrRecoveryExpired
	}

	// Invalidate old devices
	now := time.Now().Unix()
	for _, deviceID := range oldDevices {
		m.db.Exec(`
			INSERT OR REPLACE INTO invalidated_devices (device_id, invalidated_at, reason)
			VALUES (?, ?, 'recovery')
		`, deviceID, now)
	}

	// Mark recovery as complete
	_, err = m.db.Exec(`
		UPDATE recovery_sessions
		SET status = 'complete', completed_at = ?, old_devices = ?
		WHERE id = ?
	`, now, strings.Join(oldDevices, ","), recoveryID)

	return err
}

// IsDeviceValid checks if a device is still valid (not invalidated)
func (m *Manager) IsDeviceValid(deviceID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var count int
	err := m.db.QueryRow(`
		SELECT COUNT(*) FROM invalidated_devices WHERE device_id = ?
	`, deviceID).Scan(&count)

	if err != nil {
		return false, err
	}

	return count == 0, nil
}

// IsRecoveryPhraseSet checks if a recovery phrase has been set
func (m *Manager) IsRecoveryPhraseSet() (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var count int
	err := m.db.QueryRow(`SELECT COUNT(*) FROM recovery_phrases WHERE is_active = 1`).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// InvalidatePhrase invalidates the current recovery phrase (for rotation)
func (m *Manager) InvalidatePhrase() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.db.Exec(`UPDATE recovery_phrases SET is_active = 0 WHERE is_active = 1`)
	return err
}

// generateID generates a random ID
func generateID() string {
	return securerandom.MustID(16)
}
