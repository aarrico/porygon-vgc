-- Core reference schema (AD-15). A table carries data_set_id only where a
-- Champions regulation demonstrably overrides it; everything else is
-- generation-scoped. Ability slots and learnsets are absent on purpose: both
-- are data_set-versioned relationships hanging off an unversioned species, and
-- land together in a later slice.

CREATE TABLE data_set (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    identifier text NOT NULL UNIQUE,
    parent_id  bigint REFERENCES data_set (id),
    -- Catches the one-step cycle only; a longer cycle is the loader's problem.
    CONSTRAINT data_set_parent_not_self CHECK (parent_id <> id)
);

CREATE TABLE generation (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    identifier text NOT NULL UNIQUE,
    -- Sort key: 'generation-ix' does not order against 'generation-viii'.
    number     smallint NOT NULL UNIQUE CHECK (number > 0)
);

CREATE TABLE type (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    identifier    text NOT NULL UNIQUE,
    generation_id bigint NOT NULL REFERENCES generation (id)
);

CREATE TABLE stat (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    identifier     text NOT NULL UNIQUE,
    -- accuracy and evasion exist only as in-battle stages, never as base stats.
    is_battle_only boolean NOT NULL
);

CREATE TABLE nature (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    identifier        text NOT NULL UNIQUE,
    -- Both NULL for the five neutral natures.
    increased_stat_id bigint REFERENCES stat (id),
    decreased_stat_id bigint REFERENCES stat (id),
    CONSTRAINT nature_modifiers_paired
        CHECK ((increased_stat_id IS NULL) = (decreased_stat_id IS NULL)),
    CONSTRAINT nature_modifiers_differ
        CHECK (increased_stat_id IS NULL OR increased_stat_id <> decreased_stat_id)
);

CREATE TABLE species (
    id                   bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    identifier           text NOT NULL UNIQUE,
    -- Not a key: regional formes share their base forme's National Dex number,
    -- so Kantonian, Alolan and Galarian Meowth are all 52.
    national_dex         smallint NOT NULL CHECK (national_dex > 0),
    is_default           boolean NOT NULL,
    generation_id        bigint NOT NULL REFERENCES generation (id),
    type_1_id            bigint NOT NULL REFERENCES type (id),
    type_2_id            bigint REFERENCES type (id),
    base_hp              smallint NOT NULL CHECK (base_hp > 0),
    base_attack          smallint NOT NULL CHECK (base_attack > 0),
    base_defense         smallint NOT NULL CHECK (base_defense > 0),
    base_special_attack  smallint NOT NULL CHECK (base_special_attack > 0),
    base_special_defense smallint NOT NULL CHECK (base_special_defense > 0),
    base_speed           smallint NOT NULL CHECK (base_speed > 0),
    CONSTRAINT species_types_differ
        CHECK (type_2_id IS NULL OR type_2_id <> type_1_id)
);

-- Exactly one default forme per National Dex number.
CREATE UNIQUE INDEX species_default_per_national_dex
    ON species (national_dex) WHERE is_default;

CREATE TABLE move (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    data_set_id   bigint NOT NULL REFERENCES data_set (id),
    identifier    text NOT NULL,
    generation_id bigint NOT NULL REFERENCES generation (id),
    type_id       bigint NOT NULL REFERENCES type (id),
    damage_class  text NOT NULL CHECK (damage_class IN ('physical', 'special', 'status')),
    -- NULL power on a status move; NULL accuracy on a move that cannot miss.
    power         smallint CHECK (power >= 0),
    accuracy      smallint CHECK (accuracy BETWEEN 1 AND 100),
    pp            smallint NOT NULL CHECK (pp > 0),
    priority      smallint NOT NULL,
    -- PokeAPI move_target identifier; open vocabulary, so no lookup table.
    target        text NOT NULL,
    UNIQUE (data_set_id, identifier),
    -- FK target for the versioned child tables below. Without data_set_id in
    -- the referenced key a child could point at a move from a different Data
    -- Set and nothing in the schema would stop it.
    UNIQUE (id, data_set_id)
);

CREATE TABLE move_meta (
    move_id        bigint PRIMARY KEY,
    data_set_id    bigint NOT NULL REFERENCES data_set (id),
    category       text NOT NULL,
    ailment        text NOT NULL,
    min_hits       smallint CHECK (min_hits > 0),
    max_hits       smallint CHECK (max_hits > 0),
    min_turns      smallint CHECK (min_turns > 0),
    max_turns      smallint CHECK (max_turns > 0),
    -- Negative drain is recoil; negative healing is self-damage.
    drain          smallint NOT NULL CHECK (drain BETWEEN -100 AND 100),
    healing        smallint NOT NULL CHECK (healing BETWEEN -100 AND 100),
    crit_rate      smallint NOT NULL CHECK (crit_rate >= 0),
    ailment_chance smallint NOT NULL CHECK (ailment_chance BETWEEN 0 AND 100),
    flinch_chance  smallint NOT NULL CHECK (flinch_chance BETWEEN 0 AND 100),
    stat_chance    smallint NOT NULL CHECK (stat_chance BETWEEN 0 AND 100),
    CONSTRAINT move_meta_hits_ordered CHECK (max_hits >= min_hits),
    CONSTRAINT move_meta_turns_ordered CHECK (max_turns >= min_turns),
    FOREIGN KEY (move_id, data_set_id) REFERENCES move (id, data_set_id)
);

CREATE TABLE move_stat_change (
    move_id     bigint NOT NULL,
    data_set_id bigint NOT NULL REFERENCES data_set (id),
    stat_id     bigint NOT NULL REFERENCES stat (id),
    change      smallint NOT NULL CHECK (change BETWEEN -6 AND 6 AND change <> 0),
    PRIMARY KEY (move_id, stat_id),
    FOREIGN KEY (move_id, data_set_id) REFERENCES move (id, data_set_id)
);

-- FR1's access path: every move matching "Attack +2 stages".
CREATE INDEX move_stat_change_effect ON move_stat_change (stat_id, change);

CREATE TABLE ability (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    data_set_id   bigint NOT NULL REFERENCES data_set (id),
    identifier    text NOT NULL,
    generation_id bigint NOT NULL REFERENCES generation (id),
    UNIQUE (data_set_id, identifier)
);

CREATE TABLE item (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    data_set_id bigint NOT NULL REFERENCES data_set (id),
    identifier  text NOT NULL,
    -- PokeAPI item_category identifier; open vocabulary, so no lookup table.
    category    text NOT NULL,
    fling_power smallint CHECK (fling_power >= 0),
    UNIQUE (data_set_id, identifier)
);
