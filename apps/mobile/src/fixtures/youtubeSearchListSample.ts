/**
 * Sample `youtube#searchListResponse` from the YouTube Data API (dev / Test tab).
 * Last `items[]` entry is a channel result — filter with `'videoId' in item.id` for embeds.
 */
export const YOUTUBE_SEARCH_LIST_SAMPLE = {
  kind: 'youtube#searchListResponse',
  etag: 'wsU2RXA3TGg5XMQfTqrAQfXpJ2c',
  nextPageToken: 'CAUQAA',
  regionCode: 'US',
  pageInfo: {
    totalResults: 1000000,
    resultsPerPage: 5,
  },
  items: [
    {
      kind: 'youtube#searchResult',
      etag: 'kCWyYgfiRyh94D4pD7UpUTykmxA',
      id: {
        kind: 'youtube#video',
        videoId: 'BKhrWiVhJ9g',
      },
      snippet: {
        publishedAt: '2026-04-22T22:07:55Z',
        channelId: 'UC0WYRNtJAVUxSA9iP7kIoTA',
        title:
          '🔴 BARCELONA X CELTA AO VIVO AGORA 🔥 LA LIGA | JOGO AO VIVO COM IMAGENS DO CAMP NOU',
        description:
          'Barcelona X Celta ao vivo. JOGO AO VIVO AGORA. Transmissão ao vivo direto do Spotify Camp Nou - La Liga 2025/26 - 33ª ...',
        thumbnails: {
          default: {
            url: 'https://i.ytimg.com/vi/BKhrWiVhJ9g/default.jpg',
            width: 120,
            height: 90,
          },
          medium: {
            url: 'https://i.ytimg.com/vi/BKhrWiVhJ9g/mqdefault.jpg',
            width: 320,
            height: 180,
          },
          high: {
            url: 'https://i.ytimg.com/vi/BKhrWiVhJ9g/hqdefault.jpg',
            width: 480,
            height: 360,
          },
        },
        channelTitle: 'Na Rede Oficial',
        liveBroadcastContent: 'none',
        publishTime: '2026-04-22T22:07:55Z',
      },
    },
    {
      kind: 'youtube#searchResult',
      etag: 'uq3rIj04qWSVuPmWnkYNntiS8MM',
      id: {
        kind: 'youtube#video',
        videoId: 'jjKbkidgR7c',
      },
      snippet: {
        publishedAt: '2026-04-22T22:05:20Z',
        channelId: 'UCiN40zpNQVWT9cUeahegtmA',
        title:
          '✅ BARCELONA VENCIÓ a CELTA y dio un PASE CLAVE en LA LIGA: LAMINE YAMAL PIDIÓ el CAMBIO por LESIÓN',
        description:
          'barcelona #celtadevigo #envivo #laliga Relata: Franco Lattuca Comenta: Marcos Pelayo . . . . Barcelona vs Celta de Vigo en ...',
        thumbnails: {
          default: {
            url: 'https://i.ytimg.com/vi/jjKbkidgR7c/default.jpg',
            width: 120,
            height: 90,
          },
          medium: {
            url: 'https://i.ytimg.com/vi/jjKbkidgR7c/mqdefault.jpg',
            width: 320,
            height: 180,
          },
          high: {
            url: 'https://i.ytimg.com/vi/jjKbkidgR7c/hqdefault.jpg',
            width: 480,
            height: 360,
          },
        },
        channelTitle: 'Cábala Futbolera',
        liveBroadcastContent: 'none',
        publishTime: '2026-04-22T22:05:20Z',
      },
    },
    {
      kind: 'youtube#searchResult',
      etag: 'q4y9bF_qKV2d5ShLbfbmEEq3Fmk',
      id: {
        kind: 'youtube#video',
        videoId: 'SobAeCTIe_o',
      },
      snippet: {
        publishedAt: '2026-04-22T22:05:06Z',
        channelId: 'UCBPasXWxj1DYJLqZktw_UWg',
        title:
          '✅ BARCELONA vs CELTA DE VIGO EN VIVO LIVE EN ESPAÑOL 🏆 JUEGA LAMINE YAMAL LA LIGA ESPAÑOLA',
        description:
          'BARCELONA vs CELTA DE VIGO EN VIVO LIVE EN ESPAÑOL JUEGA LAMINE YAMAL LA LIGA ESPAÑOLA DONÁ CON ...',
        thumbnails: {
          default: {
            url: 'https://i.ytimg.com/vi/SobAeCTIe_o/default.jpg',
            width: 120,
            height: 90,
          },
          medium: {
            url: 'https://i.ytimg.com/vi/SobAeCTIe_o/mqdefault.jpg',
            width: 320,
            height: 180,
          },
          high: {
            url: 'https://i.ytimg.com/vi/SobAeCTIe_o/hqdefault.jpg',
            width: 480,
            height: 360,
          },
        },
        channelTitle: 'Secta Deportiva',
        liveBroadcastContent: 'none',
        publishTime: '2026-04-22T22:05:06Z',
      },
    },
    {
      kind: 'youtube#searchResult',
      etag: '7HXhyiumikFtbqsBf5pRr5QKaz4',
      id: {
        kind: 'youtube#video',
        videoId: 'dAtynmFL0Ss',
      },
      snippet: {
        publishedAt: '2026-04-22T22:00:43Z',
        channelId: 'UCWmsMLh21U35lmlwdrDTo6g',
        title:
          'LAMINE NEM CONSEGUIU COMEMORAR seu GOL! #lamineyaml #lamine #yamal #barcelona',
        description: '',
        thumbnails: {
          default: {
            url: 'https://i.ytimg.com/vi/dAtynmFL0Ss/default.jpg',
            width: 120,
            height: 90,
          },
          medium: {
            url: 'https://i.ytimg.com/vi/dAtynmFL0Ss/mqdefault.jpg',
            width: 320,
            height: 180,
          },
          high: {
            url: 'https://i.ytimg.com/vi/dAtynmFL0Ss/hqdefault.jpg',
            width: 480,
            height: 360,
          },
        },
        channelTitle: 'Canal DPSA - Notícias e Futebol',
        liveBroadcastContent: 'none',
        publishTime: '2026-04-22T22:00:43Z',
      },
    },
    {
      kind: 'youtube#searchResult',
      etag: 'qnhkUheY6raiC8pQ2xxozmQYiOU',
      id: {
        kind: 'youtube#video',
        videoId: 'OIWSBZogs8I',
      },
      snippet: {
        publishedAt: '2026-04-22T21:59:35Z',
        channelId: 'UC8dwL-miT3zeLqPiIs0_TRA',
        title:
          '🔴EN VIVO || 🔥FC BARCELONA CELTA EN VIVO 🏆 LA LIGA EA SPORTS 25/26 JORNADA 33🏆BARÇA HOY',
        description:
          'EN VIVO || FC BARCELONA CELTA EN VIVO LA LIGA EA SPORTS 25/26 JORNADA 33  BARÇA HOY CELTA FC ...',
        thumbnails: {
          default: {
            url: 'https://i.ytimg.com/vi/OIWSBZogs8I/default.jpg',
            width: 120,
            height: 90,
          },
          medium: {
            url: 'https://i.ytimg.com/vi/OIWSBZogs8I/mqdefault.jpg',
            width: 320,
            height: 180,
          },
          high: {
            url: 'https://i.ytimg.com/vi/OIWSBZogs8I/hqdefault.jpg',
            width: 480,
            height: 360,
          },
        },
        channelTitle: 'Barça Hoy',
        liveBroadcastContent: 'none',
        publishTime: '2026-04-22T21:59:35Z',
      },
    },
  ],
} as const;

export function videoItemsFromSearchSample() {
  return YOUTUBE_SEARCH_LIST_SAMPLE.items.filter(
    (
      item,
    ): item is (typeof YOUTUBE_SEARCH_LIST_SAMPLE.items)[number] & {
      id: {kind: 'youtube#video'; videoId: string};
    } => 'videoId' in item.id,
  );
}
