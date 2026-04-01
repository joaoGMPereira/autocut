package seed

// channelDefaults holds the predefined configuration for a specific YouTube channel.
// Port of PredefinedChannels.kt — maps YouTubeChannelID → defaults.
type channelDefaults struct {
	categoryID         int
	playlistIDCortes   string
	playlistIDShorts   string
	tags               string
	gradientColorStart string
	gradientColorEnd   string
	pinnedComment      string
}

// predefinedChannels maps YouTube channel IDs to their predefined config.
// These are kept in sync with PredefinedChannels.kt in the Kotlin app.
var predefinedChannels = map[string]channelDefaults{
	// React / Programação
	"UCSwTDK61hsBFV4CToQtFl-g": {
		categoryID:         28,
		playlistIDCortes:   "PLz4FQFPuKNaz7blXwx_pgNBdk8KWEzWwC",
		playlistIDShorts:   "PLz4FQFPuKNazfULbLOaiwPykDBBhzRrZW",
		tags:               "javascript,canvas,gemini,html,programação,css,vibecoding,gameplay,gamedev,jogo no navegador,ia,ia vs ia,codando com ia,criando jogos com ia,google gemini,inteligência artificial,chatgpt,desenvolvimento de jogos,jogos com ia,do zero",
		gradientColorStart: "#FAF98C",
		gradientColorEnd:   "#176BFF",
		pinnedComment:      "Se inscreva no canal, comente e deixe o like! ❤️",
	},
	// Maromba / Bodybuilding
	"UCpGpj9G3W1ofZGG2kcIWvnQ": {
		categoryID:         17,
		playlistIDCortes:   "PL1gfq1_qdCBAOCCaQpuLtO-Byq65B6up8",
		playlistIDShorts:   "PL1gfq1_qdCBCdTiDk3YxVC8WKxXe187ZM",
		tags:               "renato cariani,bodybuilding,bodybuilder,fisiologia,farmacia,farmacologia,treino,musculação,educação fisica,quimica,cariane,emagrecer,dieta,alimentação,saudavel,natural,suplemento,perder peso,receita,fit,toguro,mansão maromba,sabor,sabor energético",
		gradientColorStart: "#C80E0E",
		gradientColorEnd:   "#0F0F0F",
		pinnedComment:      "Se inscreva no canal, comente e deixe o like! ❤️",
	},
	// iNerd / Entretenimento
	"UCpjSuvMTQAq6SKuRE_SoaDQ": {
		categoryID:         24,
		playlistIDCortes:   "PLKj6HM7XxhZnZhjwjuhTTZsuMdxvciYLT",
		playlistIDShorts:   "PLKj6HM7XxhZlwC-6lHg084F39b8RuaYDJ",
		tags:               "nerd,ei nerd,peter jordan,ucm,marvel,dc,dc comics,cinema,filmes,animes,quadrinhos,hqs,series,geek,cultura pop,seriados,heróis,super herois,peteraqui,einerdtv,cortes peter jordan,cortes einerdtv,cortes ei nerd,canal do peter jordan,canal ei nerd,reação do peter jordan,peter jordan reagindo,react peter jordan,react ei nerd,quadro ei nerd,nerdices do peter jordan,opinião do peter jordan,análise do peter jordan,crítica do peter jordan",
		gradientColorStart: "#FAE70D",
		gradientColorEnd:   "#F43414",
		pinnedComment:      "Se inscreva no canal, comente e deixe o like! ❤️",
	},
	// React / Cortes (brino)
	"UCNcaOXuxyjSwrbEhgZWA-NA": {
		categoryID:         24,
		playlistIDCortes:   "PLnpzTyt6JdHJ6vSWuiIvv5BXxcdKyi8QY",
		playlistIDShorts:   "PLnpzTyt6JdHKUc4qiASAug0VkSGgFOB2U",
		tags:               "brino,react,engraçado,brunozor,bruninzor,reagindo,assistindo,bruninho,brinozor,videos engraçados,video react,experimentando,comida,viagem,humor,curiosidade,tiktoks,história,entretenimento,luxo",
		gradientColorStart: "#F27121",
		gradientColorEnd:   "#8A2387",
		pinnedComment:      "Se inscreva no canal, comente e deixe o like! ❤️",
	},
}
