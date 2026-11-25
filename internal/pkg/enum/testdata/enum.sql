create schema "ref";

create table ref."event_type"
(
    "id"    smallint primary key,
    "sid"   varchar(31) not null,
    "label" varchar(255)
);

insert into ref."event_type" ("id", "sid", "label")
values (1, 'r1', 'RDV d''estimation'),
       (6, 'conge', 'Congé'),
       (17, 'companyEvent', 'Événement d’entreprise'),
       (28, 'relance R1', 'Relance R1');

create table ref."empty_type"
(
    "id"    integer primary key,
    "label" text
);

create table ref."text_id"
(
    "id"    text primary key,
    "label" text
);

insert into ref."text_id" ("id", "label")
values ('a', 'A');

-- a label that is NULL in the database, read through the driver
create table ref."null_label"
(
    "id"      integer primary key,
    "id_name" text,
    "label"   text
);

insert into ref."null_label" ("id", "id_name", "label")
values (3, 'trois', NULL);
