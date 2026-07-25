-- +goose Up
-- Seed all 30 MLS clubs into team_youtube_channels.
-- Channel IDs resolved via YouTube Data API channels.list?forHandle=… on 2026-07-25.

INSERT INTO team_youtube_channels (lookup_key, country, channel_id, channel_handle, is_verified)
VALUES
    -- Eastern Conference
    ('inter miami', 'USA', 'UCcFuqX01k0dCBULXl1YvCsA', '@intermiamicf', true),
    ('atlanta united', 'USA', 'UC8fg8L4X7qpQdHJgxpM4qxw', '@AtlantaUnited', true),
    ('charlotte fc', 'USA', 'UCx4A5iThing38J7V05aLL2g', '@Charlotte.F.C', true),
    ('chicago fire', 'USA', 'UCMDsGMFC8boMU4I0nNC5bIw', '@chicagofire', true),
    ('cf montreal', 'Canada', 'UC7rAKewGgSTTJ7oEjwv5jzA', '@cfmontreal', true),
    ('columbus crew', 'USA', 'UCWaD_J0qRubgPr9DV_x9-Jw', '@ColumbusCrew', true),
    ('dc united', 'USA', 'UCsnTNtz7kUQNomCGlCQG-AA', '@dcunited', true),
    ('fc cincinnati', 'USA', 'UCqT7uS7skbqqNrn5LCCozMg', '@FCCincinnati', true),
    ('new england revolution', 'USA', 'UCn1LnRzToob6tRp8V_ijeBQ', '@nerevolution', true),
    ('new york city fc', 'USA', 'UCmZQ-7cZhtXz8eH6UJza4JA', '@newyorkcityfc', true),
    ('new york red bulls', 'USA', 'UCOehRhbXyxRGsNO0nkLgZfA', '@redbullnewyork', true),
    ('orlando city', 'USA', 'UCrCHG9q2F2U57_QSEKHWvIQ', '@OrlandoCitySC', true),
    ('philadelphia union', 'USA', 'UCa27XgnHniWKfXcEDGfwIgA', '@PhiladelphiaUnion', true),
    ('toronto fc', 'Canada', 'UCc1qA_64TEqAX9pnGJOO0Dw', '@TorontoFC', true),
    ('nashville sc', 'USA', 'UCoyHn37VOAANnjxQjL4LARA', '@Nashvillesc', true),
    -- Western Conference
    ('la galaxy', 'USA', 'UCQsasqjpe_blMBHbUoWYGZg', '@lagalaxy', true),
    ('lafc', 'USA', 'UCnqj91wjXAT0bsmxF1fHWBA', '@LAFC', true),
    ('austin fc', 'USA', 'UC7u24YKDCkuioxWy5hTeTdQ', '@AustinFC', true),
    ('colorado rapids', 'USA', 'UCJCvuIrw1gAfQnTJRLGrPDQ', '@ColoradoRapids', true),
    ('fc dallas', 'USA', 'UCE2JDtOSy6R-9dwkGIWgyEA', '@FCDallas', true),
    ('houston dynamo', 'USA', 'UCMoXjcnOvCwUUm4M6ePY7Gg', '@houstondynamo', true),
    ('minnesota united', 'USA', 'UCnFoCifXR_Qp5iXhK73-kjQ', '@MNUnitedFC', true),
    ('portland timbers', 'USA', 'UCm0KnY18KTa_h9bcm3aFEBw', '@TimbersFC', true),
    ('real salt lake', 'USA', 'UCGYzqSVKpc7svCtDbb-uEgw', '@realsaltlake', true),
    ('san jose earthquakes', 'USA', 'UCw1I_b0PbB67iqNGdos0HYw', '@sjearthquakes', true),
    ('seattle sounders', 'USA', 'UCVhbRUhe_hfmgi-UN1gcQzw', '@SoundersFC', true),
    ('sporting kc', 'USA', 'UCDcDw3eCZvwNQdLfSUzhkTg', '@SportingKC', true),
    ('st louis city', 'USA', 'UCNxHd0AvwN7uhyAh7ZpRsdg', '@stlcitysc', true),
    ('vancouver whitecaps', 'Canada', 'UCJ_qxVONy87ISlB2sYwIlvA', '@WhitecapsFC', true),
    ('san diego fc', 'USA', 'UCOJ8WBGCN2kxectMgoypybg', '@SanDiegoFC', true)
ON CONFLICT (lookup_key) DO UPDATE SET
    channel_id = EXCLUDED.channel_id,
    channel_handle = EXCLUDED.channel_handle,
    country = EXCLUDED.country,
    is_verified = EXCLUDED.is_verified,
    updated_at = NOW();

-- +goose Down

DELETE FROM team_youtube_channels
WHERE lookup_key IN (
    'inter miami',
    'atlanta united',
    'charlotte fc',
    'chicago fire',
    'cf montreal',
    'columbus crew',
    'dc united',
    'fc cincinnati',
    'new england revolution',
    'new york city fc',
    'new york red bulls',
    'orlando city',
    'philadelphia union',
    'toronto fc',
    'nashville sc',
    'la galaxy',
    'lafc',
    'austin fc',
    'colorado rapids',
    'fc dallas',
    'houston dynamo',
    'minnesota united',
    'portland timbers',
    'real salt lake',
    'san jose earthquakes',
    'seattle sounders',
    'sporting kc',
    'st louis city',
    'vancouver whitecaps',
    'san diego fc'
);
