create table titles (
	id varchar(12) not null,
	year integer not null,
	title varchar(2048) not null,
	updated integer,
	rating real,
	play_url varchar(2048),
	synopsis text,
	box_art varchar(2048),
	constraint pk_id primary key (id)
);
