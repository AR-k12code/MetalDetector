--
-- PostgreSQL database dump
--

-- Dumped from database version 13.2
-- Dumped by pg_dump version 13.2

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: case_insensitive; Type: COLLATION; Schema: public; Owner: postgres
--

CREATE COLLATION public.case_insensitive (provider = icu, deterministic = false, locale = 'en-US-u-ks-level2');


ALTER COLLATION public.case_insensitive OWNER TO postgres;

--
-- Name: ltree; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS ltree WITH SCHEMA public;


--
-- Name: EXTENSION ltree; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION ltree IS 'data type for hierarchical tree-like structures';


--
-- Name: postgis; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS postgis WITH SCHEMA public;


--
-- Name: EXTENSION postgis; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION postgis IS 'PostGIS geometry and geography spatial types and functions';


--
-- Name: asset_status; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.asset_status AS ENUM (
    'DEPLOYED',
    'RETIRED',
    'STOLEN',
    'BROKEN',
    'UNASSIGNED',
    'DISABLED',
    'LOST'
);


ALTER TYPE public.asset_status OWNER TO postgres;

--
-- Name: chromebook_status; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.chromebook_status AS ENUM (
    'ACTIVE',
    'DELINQUENT',
    'DEPROVISIONED',
    'DISABLED',
    'INACTIVE',
    'RETURN_ARRIVED',
    'RETURN_REQUESTED',
    'SHIPPED',
    'UNKNOWN'
);


ALTER TYPE public.chromebook_status OWNER TO postgres;

--
-- Name: grade; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.grade AS ENUM (
    'PRE-K',
    'KINDERGARTEN',
    '1',
    '2',
    '3',
    '4',
    '5',
    '6',
    '7',
    '8',
    '9',
    '10',
    '11',
    '12',
    'SUPER-SENIOR',
    'GRADUATED',
    'SM'
);


ALTER TYPE public.grade OWNER TO postgres;

--
-- Name: student_status; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.student_status AS ENUM (
    'ACTIVE',
    'GRADUATED',
    'INACTIVE',
    'PRE-REGISTERED'
);


ALTER TYPE public.student_status OWNER TO postgres;

--
-- Name: insert_ping(inet, uuid, integer, integer, text, inet, inet, real, real, real, integer, text); Type: PROCEDURE; Schema: public; Owner: postgres
--

CREATE PROCEDURE public.insert_ping(request_ip inet DEFAULT NULL::inet, device_id uuid DEFAULT NULL::uuid, client_time integer DEFAULT NULL::integer, session_start integer DEFAULT NULL::integer, serial text DEFAULT NULL::text, local_ipv4 inet DEFAULT NULL::inet, local_ipv6 inet DEFAULT NULL::inet, latitude real DEFAULT NULL::real, longitude real DEFAULT NULL::real, accuracy real DEFAULT NULL::real, location_time integer DEFAULT NULL::integer, email text DEFAULT NULL::text)
    LANGUAGE sql
    AS $_$
UPDATE pings_raw
SET latest_for_device = false
WHERE
	latest_for_session = true AND
	serial = $5;

UPDATE pings_raw
SET latest_for_user = false
WHERE
	latest_for_session = true AND
	email = $12;

UPDATE pings_raw
SET latest_for_session = false
WHERE
	latest_for_session = true AND
	serial = $5 AND
	email = $12 AND
	session_start = $4;

INSERT INTO pings_raw (
	request_ip,
	device_id,
	client_time,
	session_start,
	serial,
	local_ipv4,
	local_ipv6,
	latitude,
	longitude,
	accuracy,
	location_time,
	email,
	closest_building,
	latest_for_device,
	latest_for_user,
	latest_for_session
) VALUES (
	request_ip,
	device_id,
	client_time,
	session_start,
	serial,
	local_ipv4,
	local_ipv6,
	latitude,
	longitude,
	accuracy,
	location_time,
	email,
	CASE
		WHEN latitude IS NULL OR longitude IS NULL THEN NULL
		ELSE (
			SELECT buildings.abbreviation
			FROM buildings
			ORDER BY (ST_Distance(ST_Makepoint(longitude, latitude)::geography, buildings.location))
			LIMIT 1
		)
	END,
	true,
	true,
	true
);
$_$;


ALTER PROCEDURE public.insert_ping(request_ip inet, device_id uuid, client_time integer, session_start integer, serial text, local_ipv4 inet, local_ipv6 inet, latitude real, longitude real, accuracy real, location_time integer, email text) OWNER TO postgres;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: assets; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.assets (
    serial text NOT NULL COLLATE public.case_insensitive,
    asset_number integer,
    status public.asset_status,
    location text COLLATE public.case_insensitive,
    room text COLLATE public.case_insensitive,
    model text COLLATE public.case_insensitive,
    client text COLLATE public.case_insensitive,
    notes text
);


ALTER TABLE public.assets OWNER TO postgres;

--
-- Name: buildings; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.buildings (
    abbreviation text NOT NULL COLLATE public.case_insensitive,
    location public.geography(Point,4326)
);


ALTER TABLE public.buildings OWNER TO postgres;

--
-- Name: chromebooks; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.chromebooks (
    device_id uuid NOT NULL,
    serial text NOT NULL COLLATE public.case_insensitive,
    status public.chromebook_status,
    last_sync timestamp with time zone,
    "user" text COLLATE public.case_insensitive,
    location text COLLATE public.case_insensitive,
    asset_id text COLLATE public.case_insensitive,
    notes text,
    model text COLLATE public.case_insensitive,
    os_version public.ltree,
    wifi_mac macaddr,
    ethernet_mac macaddr,
    dev_mode boolean,
    enrollment_time timestamp with time zone,
    org_unit text COLLATE public.case_insensitive,
    recent_users text[] COLLATE public.case_insensitive,
    lan_ip inet,
    wan_ip inet,
    org_unit_path public.ltree GENERATED ALWAYS AS ((replace(btrim(regexp_replace((org_unit COLLATE "en-US-x-icu"), '[^A-Za-z0-9/]+'::text, '_'::text, 'g'::text), '/'::text), '/'::text, '.'::text))::public.ltree) STORED,
    url text GENERATED ALWAYS AS (('https://admin.google.com/ac/chrome/devices/'::text || (device_id)::text)) STORED
);


ALTER TABLE public.chromebooks OWNER TO postgres;

--
-- Name: pings_raw; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.pings_raw (
    server_time integer DEFAULT date_part('epoch'::text, now()),
    request_ip inet,
    device_id uuid,
    client_time integer,
    session_start integer,
    serial text COLLATE public.case_insensitive,
    local_ipv4 inet,
    local_ipv6 inet,
    latitude real,
    longitude real,
    accuracy real,
    location_time integer,
    email text COLLATE public.case_insensitive,
    closest_building text COLLATE public.case_insensitive,
    latest_for_device boolean NOT NULL,
    latest_for_user boolean NOT NULL,
    latest_for_session boolean NOT NULL
)
PARTITION BY LIST (latest_for_session);


ALTER TABLE public.pings_raw OWNER TO postgres;

--
-- Name: pings; Type: VIEW; Schema: public; Owner: postgres
--

CREATE VIEW public.pings AS
 WITH latest_for_devices AS (
         SELECT pings_raw_1.serial,
            max(pings_raw_1.server_time) AS server_time
           FROM public.pings_raw pings_raw_1
          GROUP BY pings_raw_1.serial
        ), latest_for_users AS (
         SELECT pings_raw_1.email,
            max(pings_raw_1.server_time) AS server_time
           FROM public.pings_raw pings_raw_1
          GROUP BY pings_raw_1.email
        ), latest_for_sessions AS (
         SELECT pings_raw_1.serial,
            pings_raw_1.email,
            pings_raw_1.session_start,
            max(pings_raw_1.server_time) AS server_time
           FROM public.pings_raw pings_raw_1
          GROUP BY pings_raw_1.serial, pings_raw_1.email, pings_raw_1.session_start
        )
 SELECT to_timestamp((pings_raw.server_time)) AS server_time,
    pings_raw.request_ip,
    pings_raw.device_id,
    to_timestamp((pings_raw.client_time)) AS client_time,
    to_timestamp((pings_raw.session_start)) AS session_start,
    pings_raw.serial,
    pings_raw.local_ipv4,
    pings_raw.local_ipv6,
    pings_raw.latitude,
    pings_raw.longitude,
    pings_raw.accuracy,
    to_timestamp((pings_raw.location_time)) AS location_time,
    pings_raw.email,
    pings_raw.closest_building,
    pings_raw.latest_for_device,
    pings_raw.latest_for_user,
    pings_raw.latest_for_session,
    public.st_distance((public.st_makepoint((pings_raw.longitude), (pings_raw.latitude)))::public.geography, closest_buildings.location, false) AS distance_to_school,
    (pings_raw.request_ip << '10.0.0.0/8'::inet) AS on_network,
    (pings_raw.device_id IS NOT NULL) AS authenticated,
    (to_timestamp((pings_raw.client_time)) - to_timestamp((pings_raw.location_time))) AS location_age,
    (to_timestamp((pings_raw.client_time)) - to_timestamp((pings_raw.session_start))) AS session_age
   FROM (public.pings_raw
     LEFT JOIN public.buildings closest_buildings ON ((pings_raw.closest_building = closest_buildings.abbreviation)));


ALTER TABLE public.pings OWNER TO postgres;

SET default_tablespace = compressed;

--
-- Name: pings_raw_archive; Type: TABLE; Schema: public; Owner: postgres; Tablespace: compressed
--

CREATE TABLE public.pings_raw_archive (
    server_time integer DEFAULT date_part('epoch'::text, now()),
    request_ip inet,
    device_id uuid,
    client_time integer,
    session_start integer,
    serial text COLLATE public.case_insensitive,
    local_ipv4 inet,
    local_ipv6 inet,
    latitude real,
    longitude real,
    accuracy real,
    location_time integer,
    email text COLLATE public.case_insensitive,
    closest_building text COLLATE public.case_insensitive,
    latest_for_device boolean NOT NULL,
    latest_for_user boolean NOT NULL,
    latest_for_session boolean NOT NULL
);
ALTER TABLE ONLY public.pings_raw ATTACH PARTITION public.pings_raw_archive DEFAULT;


ALTER TABLE public.pings_raw_archive OWNER TO postgres;

SET default_tablespace = '';

--
-- Name: pings_raw_latest; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.pings_raw_latest (
    server_time integer DEFAULT date_part('epoch'::text, now()),
    request_ip inet,
    device_id uuid,
    client_time integer,
    session_start integer,
    serial text COLLATE public.case_insensitive,
    local_ipv4 inet,
    local_ipv6 inet,
    latitude real,
    longitude real,
    accuracy real,
    location_time integer,
    email text COLLATE public.case_insensitive,
    closest_building text COLLATE public.case_insensitive,
    latest_for_device boolean NOT NULL,
    latest_for_user boolean NOT NULL,
    latest_for_session boolean NOT NULL
);
ALTER TABLE ONLY public.pings_raw ATTACH PARTITION public.pings_raw_latest FOR VALUES IN (true);


ALTER TABLE public.pings_raw_latest OWNER TO postgres;

--
-- Name: users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.users (
    username text NOT NULL COLLATE public.case_insensitive,
    email text COLLATE public.case_insensitive,
    first_name text COLLATE public.case_insensitive,
    middle_initial character(1) COLLATE public.case_insensitive,
    last_name text COLLATE public.case_insensitive,
    student_id integer,
    building text COLLATE public.case_insensitive,
    graduation_year integer,
    title text COLLATE public.case_insensitive,
    creation_date timestamp with time zone
);


ALTER TABLE public.users OWNER TO postgres;

--
-- Name: predictions; Type: VIEW; Schema: public; Owner: postgres
--

CREATE VIEW public.predictions AS
 SELECT DISTINCT chromebooks.device_id,
    chromebooks.serial,
    first_value(pings.email) OVER (PARTITION BY chromebooks.device_id ORDER BY pings.server_time) AS "user",
    first_value(pings.server_time) OVER (PARTITION BY chromebooks.device_id ORDER BY pings.server_time) AS last_used,
    first_value(pings.session_age) OVER (PARTITION BY chromebooks.device_id ORDER BY pings.server_time) AS session_age
   FROM ((public.pings
     JOIN public.chromebooks ON ((pings.serial = chromebooks.serial)))
     JOIN public.users ON ((pings.email = users.email)))
  WHERE (pings.latest_for_session AND (users.student_id IS NOT NULL) AND (pings.server_time >= (now() - '1 year'::interval)));


ALTER TABLE public.predictions OWNER TO postgres;

--
-- Name: students; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.students (
    student_id integer NOT NULL,
    email text COLLATE public.case_insensitive,
    first_name text COLLATE public.case_insensitive,
    middle_initial character(1) COLLATE public.case_insensitive,
    last_name text COLLATE public.case_insensitive,
    nickname text COLLATE public.case_insensitive,
    building text COLLATE public.case_insensitive,
    grade public.grade,
    status public.student_status,
    url text GENERATED ALWAYS AS (('https://eschool20.esp.k12.ar.us/eSchoolPLUS20/Student/Registration/StudentSummary?studentId='::text || (student_id)::text)) STORED
);


ALTER TABLE public.students OWNER TO postgres;

--
-- Name: assets assets_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT assets_pkey PRIMARY KEY (serial);


--
-- Name: buildings buildings_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.buildings
    ADD CONSTRAINT buildings_pkey PRIMARY KEY (abbreviation);


--
-- Name: chromebooks chromebooks_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chromebooks
    ADD CONSTRAINT chromebooks_pkey PRIMARY KEY (device_id);


--
-- Name: students students_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.students
    ADD CONSTRAINT students_pkey PRIMARY KEY (student_id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (username);


--
-- Name: chromebooks_serial; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX chromebooks_serial ON public.chromebooks USING hash (serial);


--
-- Name: pings_raw_email; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX pings_raw_email ON ONLY public.pings_raw USING btree (email);


--
-- Name: pings_raw_archive_email_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX pings_raw_archive_email_idx ON public.pings_raw_archive USING btree (email);


--
-- Name: pings_raw_serial; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX pings_raw_serial ON ONLY public.pings_raw USING btree (serial);


--
-- Name: pings_raw_archive_serial_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX pings_raw_archive_serial_idx ON public.pings_raw_archive USING btree (serial);


--
-- Name: pings_raw_server_time; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX pings_raw_server_time ON ONLY public.pings_raw USING brin (server_time) WITH (pages_per_range='1');


--
-- Name: pings_raw_archive_server_time_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX pings_raw_archive_server_time_idx ON public.pings_raw_archive USING brin (server_time) WITH (pages_per_range='1');


--
-- Name: pings_raw_latest_email_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX pings_raw_latest_email_idx ON public.pings_raw_latest USING btree (email);


--
-- Name: pings_raw_latest_serial_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX pings_raw_latest_serial_idx ON public.pings_raw_latest USING btree (serial);


--
-- Name: pings_raw_latest_server_time_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX pings_raw_latest_server_time_idx ON public.pings_raw_latest USING brin (server_time) WITH (pages_per_range='1');


--
-- Name: pings_raw_archive_email_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.pings_raw_email ATTACH PARTITION public.pings_raw_archive_email_idx;


--
-- Name: pings_raw_archive_serial_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.pings_raw_serial ATTACH PARTITION public.pings_raw_archive_serial_idx;


--
-- Name: pings_raw_archive_server_time_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.pings_raw_server_time ATTACH PARTITION public.pings_raw_archive_server_time_idx;


--
-- Name: pings_raw_latest_email_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.pings_raw_email ATTACH PARTITION public.pings_raw_latest_email_idx;


--
-- Name: pings_raw_latest_serial_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.pings_raw_serial ATTACH PARTITION public.pings_raw_latest_serial_idx;


--
-- Name: pings_raw_latest_server_time_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.pings_raw_server_time ATTACH PARTITION public.pings_raw_latest_server_time_idx;


--
-- Name: pings_raw pings_raw_closest_building_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE public.pings_raw
    ADD CONSTRAINT pings_raw_closest_building_fkey FOREIGN KEY (closest_building) REFERENCES public.buildings(abbreviation) MATCH FULL ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: SCHEMA public; Type: ACL; Schema: -; Owner: postgres
--

REVOKE ALL ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO metaldetector;
GRANT USAGE ON SCHEMA public TO ro_user;


--
-- Name: TABLE assets; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.assets TO ro_user;
GRANT SELECT,INSERT,TRUNCATE ON TABLE public.assets TO metaldetector;


--
-- Name: TABLE buildings; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.buildings TO ro_user;
GRANT SELECT ON TABLE public.buildings TO metaldetector;


--
-- Name: TABLE chromebooks; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT,INSERT,TRUNCATE,UPDATE ON TABLE public.chromebooks TO metaldetector;
GRANT SELECT ON TABLE public.chromebooks TO ro_user;


--
-- Name: TABLE pings_raw; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT,INSERT,UPDATE ON TABLE public.pings_raw TO metaldetector;


--
-- Name: TABLE pings; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.pings TO ro_user;


--
-- Name: TABLE users; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.users TO ro_user;
GRANT SELECT,INSERT,TRUNCATE ON TABLE public.users TO metaldetector;


--
-- Name: TABLE predictions; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.predictions TO ro_user;
GRANT SELECT ON TABLE public.predictions TO metaldetector;


--
-- Name: TABLE students; Type: ACL; Schema: public; Owner: postgres
--

GRANT INSERT,TRUNCATE ON TABLE public.students TO metaldetector;
GRANT SELECT ON TABLE public.students TO ro_user;


--
-- PostgreSQL database dump complete
--
