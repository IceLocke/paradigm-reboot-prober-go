--
-- PostgreSQL database dump
--

\restrict AS2YebXqDsCWWsFw3BIFIPAtBgTcPIuKfok2LH2jf6eUqxiwVQD3ZmJHHgMt2ap

-- Dumped from database version 16.13 (Ubuntu 16.13-0ubuntu0.24.04.1)
-- Dumped by pg_dump version 16.13 (Ubuntu 16.13-0ubuntu0.24.04.1)

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

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: best50_trend; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.best50_trend (
    b50_trend_id integer NOT NULL,
    username character varying NOT NULL,
    b50rating double precision NOT NULL,
    record_time timestamp without time zone NOT NULL,
    is_valid boolean NOT NULL
);


ALTER TABLE public.best50_trend OWNER TO postgres;

--
-- Name: best50_trend_b50_trend_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.best50_trend_b50_trend_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.best50_trend_b50_trend_id_seq OWNER TO postgres;

--
-- Name: best50_trend_b50_trend_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.best50_trend_b50_trend_id_seq OWNED BY public.best50_trend.b50_trend_id;


--
-- Name: best_play_records; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.best_play_records (
    best_record_id integer NOT NULL,
    play_record_id integer NOT NULL
);


ALTER TABLE public.best_play_records OWNER TO postgres;

--
-- Name: best_play_records_best_record_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.best_play_records_best_record_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.best_play_records_best_record_id_seq OWNER TO postgres;

--
-- Name: best_play_records_best_record_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.best_play_records_best_record_id_seq OWNED BY public.best_play_records.best_record_id;


--
-- Name: difficulties; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.difficulties (
    difficulty_id integer NOT NULL,
    name character varying NOT NULL
);


ALTER TABLE public.difficulties OWNER TO postgres;

--
-- Name: difficulties_difficulty_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.difficulties_difficulty_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.difficulties_difficulty_id_seq OWNER TO postgres;

--
-- Name: difficulties_difficulty_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.difficulties_difficulty_id_seq OWNED BY public.difficulties.difficulty_id;


--
-- Name: play_records; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.play_records (
    play_record_id integer NOT NULL,
    song_level_id integer NOT NULL,
    record_time timestamp without time zone NOT NULL,
    username character varying NOT NULL,
    score integer NOT NULL,
    rating double precision NOT NULL
);


ALTER TABLE public.play_records OWNER TO postgres;

--
-- Name: play_records_play_record_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.play_records_play_record_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.play_records_play_record_id_seq OWNER TO postgres;

--
-- Name: play_records_play_record_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.play_records_play_record_id_seq OWNED BY public.play_records.play_record_id;


--
-- Name: prober_users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.prober_users (
    user_id integer NOT NULL,
    username character varying NOT NULL,
    nickname character varying NOT NULL,
    encoded_password character varying NOT NULL,
    email character varying NOT NULL,
    qq_number integer,
    account character varying,
    account_number integer,
    uuid character varying,
    anonymous_probe boolean NOT NULL,
    upload_token character varying NOT NULL,
    is_active boolean NOT NULL,
    is_admin boolean NOT NULL
);


ALTER TABLE public.prober_users OWNER TO postgres;

--
-- Name: prober_users_user_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.prober_users_user_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.prober_users_user_id_seq OWNER TO postgres;

--
-- Name: prober_users_user_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.prober_users_user_id_seq OWNED BY public.prober_users.user_id;


--
-- Name: song_levels; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.song_levels (
    song_level_id integer NOT NULL,
    song_id integer NOT NULL,
    difficulty_id integer NOT NULL,
    level double precision NOT NULL,
    fitting_level double precision,
    level_design character varying,
    notes integer NOT NULL
);


ALTER TABLE public.song_levels OWNER TO postgres;

--
-- Name: song_levels_song_level_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.song_levels_song_level_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.song_levels_song_level_id_seq OWNER TO postgres;

--
-- Name: song_levels_song_level_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.song_levels_song_level_id_seq OWNED BY public.song_levels.song_level_id;


--
-- Name: songs; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.songs (
    song_id integer NOT NULL,
    title character varying NOT NULL,
    artist character varying NOT NULL,
    genre character varying NOT NULL,
    cover character varying NOT NULL,
    illustrator character varying NOT NULL,
    version character varying NOT NULL,
    b15 boolean NOT NULL,
    album character varying NOT NULL,
    bpm character varying NOT NULL,
    length character varying NOT NULL,
    wiki_id character varying
);


ALTER TABLE public.songs OWNER TO postgres;

--
-- Name: songs_song_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.songs_song_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.songs_song_id_seq OWNER TO postgres;

--
-- Name: songs_song_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.songs_song_id_seq OWNED BY public.songs.song_id;


--
-- Name: best50_trend b50_trend_id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.best50_trend ALTER COLUMN b50_trend_id SET DEFAULT nextval('public.best50_trend_b50_trend_id_seq'::regclass);


--
-- Name: best_play_records best_record_id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.best_play_records ALTER COLUMN best_record_id SET DEFAULT nextval('public.best_play_records_best_record_id_seq'::regclass);


--
-- Name: difficulties difficulty_id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.difficulties ALTER COLUMN difficulty_id SET DEFAULT nextval('public.difficulties_difficulty_id_seq'::regclass);


--
-- Name: play_records play_record_id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.play_records ALTER COLUMN play_record_id SET DEFAULT nextval('public.play_records_play_record_id_seq'::regclass);


--
-- Name: prober_users user_id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.prober_users ALTER COLUMN user_id SET DEFAULT nextval('public.prober_users_user_id_seq'::regclass);


--
-- Name: song_levels song_level_id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.song_levels ALTER COLUMN song_level_id SET DEFAULT nextval('public.song_levels_song_level_id_seq'::regclass);


--
-- Name: songs song_id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.songs ALTER COLUMN song_id SET DEFAULT nextval('public.songs_song_id_seq'::regclass);


--
-- Name: best50_trend best50_trend_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.best50_trend
    ADD CONSTRAINT best50_trend_pkey PRIMARY KEY (b50_trend_id);


--
-- Name: best_play_records best_play_records_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.best_play_records
    ADD CONSTRAINT best_play_records_pkey PRIMARY KEY (best_record_id);


--
-- Name: difficulties difficulties_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.difficulties
    ADD CONSTRAINT difficulties_pkey PRIMARY KEY (difficulty_id);


--
-- Name: play_records play_records_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.play_records
    ADD CONSTRAINT play_records_pkey PRIMARY KEY (play_record_id);


--
-- Name: prober_users prober_users_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.prober_users
    ADD CONSTRAINT prober_users_pkey PRIMARY KEY (user_id);


--
-- Name: prober_users prober_users_username_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.prober_users
    ADD CONSTRAINT prober_users_username_key UNIQUE (username);


--
-- Name: song_levels song_levels_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.song_levels
    ADD CONSTRAINT song_levels_pkey PRIMARY KEY (song_level_id);


--
-- Name: songs songs_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.songs
    ADD CONSTRAINT songs_pkey PRIMARY KEY (song_id);


--
-- Name: songs wiki_id_unique_constraint; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.songs
    ADD CONSTRAINT wiki_id_unique_constraint UNIQUE (wiki_id);


--
-- Name: best50_trend best50_trend_username_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.best50_trend
    ADD CONSTRAINT best50_trend_username_fkey FOREIGN KEY (username) REFERENCES public.prober_users(username);


--
-- Name: best_play_records best_play_records_play_record_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.best_play_records
    ADD CONSTRAINT best_play_records_play_record_id_fkey FOREIGN KEY (play_record_id) REFERENCES public.play_records(play_record_id);


--
-- Name: play_records play_records_song_level_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.play_records
    ADD CONSTRAINT play_records_song_level_id_fkey FOREIGN KEY (song_level_id) REFERENCES public.song_levels(song_level_id);


--
-- Name: play_records play_records_username_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.play_records
    ADD CONSTRAINT play_records_username_fkey FOREIGN KEY (username) REFERENCES public.prober_users(username);


--
-- Name: song_levels song_levels_difficulty_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.song_levels
    ADD CONSTRAINT song_levels_difficulty_id_fkey FOREIGN KEY (difficulty_id) REFERENCES public.difficulties(difficulty_id);


--
-- Name: song_levels song_levels_song_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.song_levels
    ADD CONSTRAINT song_levels_song_id_fkey FOREIGN KEY (song_id) REFERENCES public.songs(song_id);


--
-- PostgreSQL database dump complete
--

\unrestrict AS2YebXqDsCWWsFw3BIFIPAtBgTcPIuKfok2LH2jf6eUqxiwVQD3ZmJHHgMt2ap

