package debugging

import "github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"

// TileAnnotation describes atlas browsing labels, not gameplay rules. Animation
// labels describe authored sprite groups, not generated pixel-effect frame counts.
//
//nolint:tagliatelle // The developer reference dump retains its established property names.
type TileAnnotation struct {
	Description          string `json:"Description"`
	IsPartOfAnimation    bool   `json:"IsPartOfAnimation"`
	TotalAnimationFrames int    `json:"TotalAnimationFrames"`
	IsEnemy              bool   `json:"IsEnemy"`
	IsNPC                bool   `json:"IsNPC"`
	IsBuilding           bool   `json:"IsBuilding"`
}

// TileAtlasAnnotation returns a detached annotation. Missing entries have no annotation.
func TileAtlasAnnotation(index indexes.SpriteIndex) TileAnnotation {
	return tileAtlasAnnotations[index]
}

//nolint:gochecknoglobals,exhaustruct,goconst // Immutable atlas labels intentionally repeat; omitted properties are empty.
var tileAtlasAnnotations = map[indexes.SpriteIndex]TileAnnotation{
	-1:  {Description: "Do Not Draw"},                                      // NoSprite
	1:   {IsPartOfAnimation: true, TotalAnimationFrames: 2},                // Water1
	2:   {IsPartOfAnimation: true, TotalAnimationFrames: 2},                // Water2
	6:   {Description: "Smallest forest"},                                  // Forest1
	8:   {Description: "2nd smallest forest"},                              // Forest2
	9:   {Description: "2nd largest forest"},                               // Forest3
	10:  {Description: "largest forest"},                                   // Forest4
	16:  {IsBuilding: true},                                                // Hut
	17:  {IsBuilding: true},                                                // Codex
	18:  {IsBuilding: true},                                                // Keep
	19:  {IsBuilding: true},                                                // Village
	20:  {IsBuilding: true},                                                // SmallCastle
	21:  {IsBuilding: true},                                                // LargeCastle
	22:  {IsBuilding: true},                                                // CaveEntrance
	23:  {IsBuilding: true},                                                // MineEntrance
	24:  {IsBuilding: true},                                                // DoomEntrance
	25:  {IsBuilding: true},                                                // Shrine
	26:  {IsBuilding: true},                                                // BrokenShrine
	27:  {IsBuilding: true},                                                // Lighthouse
	62:  {IsBuilding: true},                                                // CastleBritianEntrace
	67:  {Description: "                                                "}, // WoodFloorShipEdge
	128: {IsPartOfAnimation: true, TotalAnimationFrames: 2},                // TortureChair1
	129: {IsPartOfAnimation: true, TotalAnimationFrames: 2},                // TortureTable2
	130: {IsPartOfAnimation: true, TotalAnimationFrames: 2},                // TortureTableWithBody1
	131: {IsPartOfAnimation: true, TotalAnimationFrames: 2},                // TortureTableWithBody2
	192: {IsPartOfAnimation: true, TotalAnimationFrames: 3, IsEnemy: true}, // ScaryBlackThing1
	193: {IsPartOfAnimation: true, TotalAnimationFrames: 3, IsEnemy: true}, // ScaryBlackThing2
	194: {IsPartOfAnimation: true, TotalAnimationFrames: 3, IsEnemy: true}, // ScaryBlackThing3
	204: {Description: "Not sure what this is"},                            // Wtf1
	205: {Description: "Not sure what this is"},                            // Wtf2
	206: {Description: "Not sure what this is"},                            // Wtf3
	207: {Description: "Not sure what this is"},                            // Wtf4
	212: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // Waterfall1
	213: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // Waterfall2
	214: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // Waterfall3
	215: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // Waterfall4
	216: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // Fountain1
	217: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // Fountain2
	218: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // Fountain3
	219: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // Fountain4
	232: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // Hourglass1
	233: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // Hourglass2
	234: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // Hourglass3
	235: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // Hourglass4
	236: {TotalAnimationFrames: 4},                                         // SnakeSign1
	237: {TotalAnimationFrames: 4},                                         // SnakeSign2
	238: {TotalAnimationFrames: 4},                                         // SnakeSign3
	239: {TotalAnimationFrames: 4},                                         // SnakeSign4
	250: {IsPartOfAnimation: true, TotalAnimationFrames: 2},                // Clock1
	251: {IsPartOfAnimation: true, TotalAnimationFrames: 2},                // Clock2
	252: {IsPartOfAnimation: true, TotalAnimationFrames: 2},                // Bellows1
	253: {IsPartOfAnimation: true, TotalAnimationFrames: 2},                // Bellows2
	300: {IsEnemy: true},                                                   // PirateShipUp
	301: {IsEnemy: true},                                                   // PirateShipRight
	302: {IsEnemy: true},                                                   // PirateShipLeft
	303: {IsEnemy: true},                                                   // PirateShipDown
	308: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // SitChairDownEat1
	309: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // SitChairDownEat2
	310: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // SitChairDownEat3
	311: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // SitChairDownEat4
	312: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // SitChairUpEat1
	313: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // SitChairUpEat2
	314: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // SitChairUpEat3
	315: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // SitChairUpEat4
	316: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // LordBritishMirror1
	317: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // LordBritishMirror2
	318: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // LordBritishMirror3
	319: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // LordBritishMirror4
	320: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Wizard1
	321: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Wizard2
	322: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Wizard3
	323: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Wizard4
	324: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Bard1
	325: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Bard2
	326: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Bard3
	327: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Bard4
	328: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Fighter1
	329: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Fighter2
	330: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Fighter3
	331: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Fighter4
	332: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Avatar1
	333: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Avatar2
	334: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Avatar3
	335: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Avatar4
	336: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // TownsPerson1
	337: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // TownsPerson2
	338: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // TownsPerson3
	339: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // TownsPerson4
	340: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Merchant1
	341: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Merchant2
	342: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Merchant3
	343: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Merchant4
	344: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Jester1
	345: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Jester2
	346: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Jester3
	347: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Jester4
	348: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // BardPlaying1
	349: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // BardPlaying2
	350: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // BardPlaying3
	351: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // BardPlaying4
	352: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // PersonStocks1
	353: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // PersonStocks2
	354: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // PersonStocks3
	355: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // PersonStocks4
	356: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // WallPrisoner1
	357: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // WallPrisoner2
	358: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // WallPrisoner3
	359: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // WallPrisoner4
	360: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Child1
	361: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Child2
	362: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Child3
	363: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Child4
	364: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Begger1
	365: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Begger2
	366: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Begger3
	367: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Begger4
	368: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Guard1
	369: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Guard2
	370: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Guard3
	371: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Guard4
	372: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Apparation1
	373: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Apparation2
	374: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Apparation3
	375: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Apparation4
	376: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Blackthorn1
	377: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Blackthorn2
	378: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Blackthorn3
	379: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // Blackthorn4
	380: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // LordBritish1
	381: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // LordBritish2
	382: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // LordBritish3
	383: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsNPC: true},   // LordBritish4
	384: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Seahorse1
	385: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Seahorse2
	386: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Seahorse3
	387: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Seahorse4
	388: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Squid1
	389: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Squid2
	390: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Squid3
	391: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Squid4
	392: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // SeaSerpent1
	393: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // SeaSerpent2
	394: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // SeaSerpent3
	395: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // SeaSerpent4
	396: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Shark1
	397: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Shark2
	398: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Shark3
	399: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Shark4
	400: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Rat1
	401: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Rat2
	402: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Rat3
	403: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Rat4
	404: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Bat1
	405: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Bat2
	406: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Bat3
	407: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Bat4
	408: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Spider1
	409: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Spider2
	410: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Spider3
	411: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Spider4
	412: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Ghost1
	413: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Ghost2
	414: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Ghost3
	415: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Ghost4
	416: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Slime1
	417: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Slime2
	418: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Slime3
	419: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Slime4
	420: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Gremlin1
	421: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Gremlin2
	422: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Gremlin3
	423: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Gremlin4
	424: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Mimic1
	425: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Mimic2
	426: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Mimic3
	427: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Mimic4
	428: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Reaper1
	429: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Reaper2
	430: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Reaper3
	431: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Reaper4
	432: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Gazer1
	433: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Gazer2
	434: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Gazer3
	435: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Gazer4
	436: {TotalAnimationFrames: -1},                                        // Shard
	437: {TotalAnimationFrames: -1},                                        // Crown
	438: {TotalAnimationFrames: -1},                                        // Sceptre
	439: {TotalAnimationFrames: -1},                                        // Amulet
	440: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // StoneGargoyle1
	441: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // StoneGargoyle2
	442: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // StoneGargoyle3
	443: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // StoneGargoyle4
	444: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // InsectSwarm1
	445: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // InsectSwarm2
	446: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // InsectSwarm3
	447: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // InsectSwarm4
	448: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Orc1
	449: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Orc2
	450: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Orc3
	451: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Orc4
	452: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Skeleton1
	453: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Skeleton2
	454: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Skeleton3
	455: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Skeleton4
	456: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Snake1
	457: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Snake2
	458: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Snake3
	459: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Snake4
	460: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Ettin1
	461: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Ettin2
	462: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Ettin3
	463: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Ettin4
	464: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Headless1
	465: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Headless2
	466: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Headless3
	467: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Headless4
	468: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Wisp1
	469: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Wisp2
	470: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Wisp3
	471: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Wisp4
	472: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Daemon1
	473: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Daemon2
	474: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Daemon3
	475: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Daemon4
	476: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Dragon1
	477: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Dragon2
	478: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Dragon3
	479: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Dragon4
	480: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // SandTrap1
	481: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // SandTrap2
	482: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // SandTrap3
	483: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // SandTrap4
	484: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Troll1
	485: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Troll2
	486: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Troll3
	487: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Troll4
	488: {TotalAnimationFrames: -1},                                        // PoisonField
	489: {TotalAnimationFrames: -1},                                        // MagicField
	490: {TotalAnimationFrames: -1},                                        // FireField
	491: {TotalAnimationFrames: -1},                                        // ElectricField
	492: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // Whirpool1
	493: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // Whirpool2
	494: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // Whirpool3
	495: {IsPartOfAnimation: true, TotalAnimationFrames: 4},                // Whirpool4
	496: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // MongBat1
	497: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // MongBat2
	498: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // MongBat3
	499: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // MongBat4
	500: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Corpser1
	501: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Corpser2
	502: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Corpser3
	503: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // Corpser4
	504: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // RotWork1
	505: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // RotWork2
	506: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // RotWork3
	507: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // RotWork4
	508: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // ShadowLord1
	509: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // ShadowLord2
	510: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // ShadowLord3
	511: {IsPartOfAnimation: true, TotalAnimationFrames: 4, IsEnemy: true}, // ShadowLord4
	517: {Description: "Square hollow highlight box"},                      // CombatHighlightBox
	518: {Description: "Square solid highlight box"},                       // FullCombatHighlightBox
	519: {Description: "Square solid highlight box"},                       // HittableCombatHighlightBox
	520: {Description: "Square solid highlight box"},                       // NotHittableCombatHighlightBox
	521: {Description: "Square box indicating attack range highlight box"}, // HittableRangeCombatHighlightBox
	522: {Description: "Arrow graphic used in combat"},                     // Arrow
	523: {Description: "Axe graphic used in combat"},                       // Axe
	524: {Description: "Sling rock graphic used in combat"},                // SlingRock
	525: {Description: "Cannonball graphic used in combat"},                // CannonBall
	526: {Description: "Hit Text Graphic"},                                 // HitBang
	527: {Description: "Missed Text Graphic"},                              // MissedBang
	528: {Description: "Grazed Text Graphic"},                              // GrazedBang
}
