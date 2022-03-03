package db

import (
	"fmt"
	"log"
	"strings"
)

var schema = `
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS plpython3u;
CREATE EXTENSION xml2;
CREATE TABLE servers(
    id serial PRIMARY KEY NOT NULL,
	uid TEXT NOT NULL DEFAULT '', 
    name text NOT NULL UNIQUE,
    username text NOT NULL DEFAULT '',
    password text NOT NULL DEFAULT '',
    auth_token text NOT NULL DEFAULT '',
    ipaddress text NOT NULL DEFAULT '',
    url text NOT NULL DEFAULT '', -- endpoint
    callback_url text NOT NULL DEFAULT '', -- url to call with response from endpoint
    http_method text NOT NULL DEFAULT 'POST',
    auth_method text NOT NULL DEFAULT '',
    allow_callbacks BOOLEAN NOT NULL DEFAULT 'f', --whether to make callbacks when destination app returns successfully
    use_ssl BOOLEAN NOT NULL DEFAULT 'f', --whether ssl is enabled for this server/app
    parse_responses BOOLEAN NOT NULL DEFAULT 't', --whether to parse responses from this server/app
    ssl_client_certkey_file TEXT NOT NULL DEFAULT '',
    start_submission_period INTEGER NOT NULL DEFAULT 0, -- starting hour for off peak period
    end_submission_period INTEGER NOT NULL DEFAULT 24, -- ending hour for off peak period
    xml_response_xpath TEXT NOT NULL DEFAULT '',
    json_response_xpath TEXT NOT NULL DEFAULT '',
    suspended BOOLEAN NOT NULL DEFAULT 'f', --whether the app, sever or endpoint is suspended
    created timestamptz DEFAULT current_timestamp,
    updated timestamptz DEFAULT current_timestamp
);

CREATE TABLE server_allowed_sources(
    id serial PRIMARY KEY NOT NULL,
    server_id INTEGER NOT NULL REFERENCES servers(id),
    allowed_sources INTEGER[] NOT NULL DEFAULT ARRAY[]::INTEGER[],
    created timestamptz DEFAULT current_timestamp,
    updated timestamptz DEFAULT current_timestamp,
    UNIQUE(server_id)
);

CREATE VIEW servers_view AS 
    SELECT a.*, b.allowed_sources FROM servers a, server_allowed_sources b where a.id = b.server_id;


CREATE TABLE requests(
    id bigserial PRIMARY KEY NOT NULL,
	uid TEXT NOT NULL DEFAULT '', 
    source INTEGER REFERENCES servers(id), -- source app/server
    destination INTEGER REFERENCES servers(id), -- source app/server
    body TEXT NOT NULL DEFAULT '',
    body_is_query_param BOOLEAN NOT NULL DEFAULT 'f',
    url_suffix TEXT DEFAULT '', -- if present, it is added to API url 
    ctype TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'ready' CHECK( status IN(
            'pending', 'ready', 'inprogress', 'failed', 'error', 'expired', 'completed', 'canceled')),
    statuscode text DEFAULT '',
    retries INTEGER NOT NULL DEFAULT 0,
    errors TEXT DEFAULT '', -- indicative response message
    submissionid INTEGER NOT NULL DEFAULT 0, -- message_id in source app -> helpful when check for already sent submissions
    period TEXT DEFAULT '', -- period
    week TEXT DEFAULT '', -- reporting week
    month TEXT DEFAULT '', -- reporting month
    year TEXT, -- year of submission
    msisdn TEXT NOT NULL DEFAULT '', -- can be report sender in source
    raw_msg TEXT NOT NULL DEFAULT '', -- raw message in source system
    facility TEXT NOT NULL DEFAULT '', -- facility owning report
    district TEXT NOT NULL DEFAULT '', -- district
    report_type TEXT NOT NULL DEFAULT '',
    extras TEXT NOT NULL DEFAULT '',
    suspended INT NOT NULL DEFAULT 0, --whether to suspend this request 0 = false, 1 = true 
    created timestamptz DEFAULT current_timestamp,
    updated timestamptz DEFAULT current_timestamp
);

CREATE INDEX requests_idx1 ON requests(submissionid);
CREATE INDEX requests_idx2 ON requests(status);
CREATE INDEX requests_idx3 ON requests(statuscode);
CREATE INDEX requests_idx4 ON requests(week);
CREATE INDEX requests_idx5 ON requests(month);
CREATE INDEX requests_idx6 ON requests(year);
CREATE INDEX requests_idx7 ON requests(ctype);
CREATE INDEX requests_idx8 ON requests(msisdn);
CREATE INDEX requests_idx9 ON requests(facility);
CREATE INDEX requests_idx10 ON requests(district);

CREATE TABLE sms_logs (
       id bigserial PRIMARY KEY,
       msg text NOT NULL,
       from_msisdn text NOT NULL,
       to_msisdn text NOT NULL,
       
       msg_len int NOT NULL DEFAULT 0,
       msg_type varchar(16) NOT NULL DEFAULT '',  
       msg_dir varchar(10) NOT NULL CHECK (msg_dir IN ('MT', 'MO')),     
       created timestamptz  NOT NULL DEFAULT current_timestamp       
);
CREATE index sms_idx1 ON sms_logs(from_msisdn);
CREATE index sms_idx2 ON sms_logs(to_msisdn);
CREATE index sms_idx3 ON sms_logs(created);

CREATE TABLE blacklist (
       id bigserial PRIMARY KEY,
       msisdn text NOT NULL,
       created timestamptz  NOT NULL DEFAULT current_timestamp,
       updated timestamptz DEFAULT current_timestamp
);
CREATE INDEX blacklist_msisdn ON blacklist(msisdn);
CREATE INDEX blacklist_created ON blacklist(created);
CREATE INDEX blacklist_updated ON blacklist(updated);

INSERT INTO servers (name, username, password, ipaddress, url, auth_method)
    VALUES
        ('localhost', 'tester', 'foobar', '127.0.0.1', 'http://localhost:8080/test', 'Basic Auth'),
        ('mtrack', 'tester', 'foobar', '127.0.0.1', 'http://localhost:8080/test', 'Basic Auth'),
        ('mtrackpro', 'tester', 'foobar', '127.0.0.1', 'http://localhost:8080/test', 'Basic Auth'),
        ('dhis2', 'tester', 'foobar', '127.0.0.1', 'http://localhost/api/dataValueSets', 'Basic Auth');
INSERT INTO server_allowed_sources (server_id, allowed_sources)
    VALUES ((SELECT id FROM servers WHERE name = 'localhost'), ARRAY[]::INTEGER[]),
    ((SELECT id FROM servers WHERE name = 'mtrack'), ARRAY[]::INTEGER[]),
    ((SELECT id FROM servers WHERE name = 'mtrackpro'), ARRAY[]::INTEGER[]);

-- Add some tables for the web interface


-- for web.py sessions
CREATE TABLE sessions (
    session_id CHAR(128) UNIQUE NOT NULL,
    atime TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    data TEXT
);

CREATE TABLE user_roles (
    id BIGSERIAL NOT NULL PRIMARY KEY,
    cdate TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    name TEXT NOT NULL UNIQUE,
    descr text DEFAULT ''
);

CREATE TABLE user_role_permissions (
    id bigserial NOT NULL PRIMARY KEY,
    user_role BIGINT NOT NULL REFERENCES user_roles ON DELETE CASCADE ON UPDATE CASCADE,
    sys_module TEXT NOT NULL, -- the name of the module - defined above this level
    sys_perms VARCHAR(16) NOT NULL,
    created timestamptz DEFAULT current_timestamp,
    updated timestamptz DEFAULT current_timestamp,
    UNIQUE(sys_module, user_role)
);

CREATE TABLE users (
    id bigserial NOT NULL PRIMARY KEY,
	uid TEXT NOT NULL DEFAULT '', 
    firstname TEXT NOT NULL,
    lastname TEXT NOT NULL,
    username TEXT NOT NULL UNIQUE,
	 telephone TEXT NOT NULL DEFAULT '',
    password TEXT NOT NULL, -- blowfish hash of password
    email TEXT,
    user_role  BIGINT NOT NULL REFERENCES user_roles ON DELETE RESTRICT ON UPDATE CASCADE,
    transaction_limit TEXT DEFAULT '0/'||to_char(NOW(),'yyyymmdd'),
    is_active BOOLEAN NOT NULL DEFAULT 't',
    is_system_user BOOLEAN NOT NULL DEFAULT 'f',
    created timestamptz DEFAULT current_timestamp,
    updated timestamptz DEFAULT current_timestamp

);

CREATE TABLE audit_log (
        id BIGSERIAL NOT NULL PRIMARY KEY,
        logtype VARCHAR(32) NOT NULL DEFAULT '',
        actor TEXT NOT NULL,
        action text NOT NULL DEFAULT '',
        remote_ip INET,
        detail TEXT NOT NULL,
        created_by INTEGER REFERENCES users(id), -- like actor id
        created TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX au_idx1 ON audit_log(created);
CREATE INDEX au_idx2 ON audit_log(logtype);
CREATE INDEX au_idx4 ON audit_log(action);

CREATE TABLE schedules(
    id bigserial NOT NULL PRIMARY KEY,
    sched_type TEXT NOT NULL DEFAULT 'sms' CHECK (sched_type IN ('sms', 'contact_push', 'url', 'command')), -- also 'push_contact'
    params JSON NOT NULL DEFAULT '{}'::json,
    sched_content TEXT, -- body of scheduled url call
    sched_url TEXT DEFAULT '',
    command TEXT DEFAULT '',
    command_args TEXT,
    first_run_at TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP, -- when to push first.
    -- should push be repeated? If so, how often?
    repeat varchar(16) NOT NULL DEFAULT 'never' CHECK (repeat IN ('never','daily','weekly','monthly','yearly')),
    last_run_at  TIMESTAMPTZ, -- when last ran
    next_run_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status text NOT NULL DEFAULT 'ready' CHECK(status IN ('ready', 'skipped', 'sent','failed','error', 'completed')),
    is_active BOOLEAN NOT NULL DEFAULT 't',
    created_by INTEGER REFERENCES users(id),
    created TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX schedules_idx1 ON schedules(created);
CREATE INDEX schedules_idx2 ON schedules(first_run_at);
CREATE INDEX schedules_idx3 ON schedules(last_run_at);
CREATE INDEX schedules_idx4 ON schedules(next_run_at);

-- FUNCTIONS
-- Check if source is an allowed 'source' for destination server/app dest
CREATE OR REPLACE FUNCTION is_allowed_source(source integer, dest integer) RETURNS BOOLEAN AS $delim$
    DECLARE
     t boolean;
    BEGIN
        select source = ANY(allowed_sources) INTO t FROM server_allowed_sources WHERE server_id = dest;
        RETURN t;
    END;
$delim$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_server_apps(xid INT) RETURNS TEXT AS
$delim$
    DECLARE
    r TEXT;
    p TEXT;
    BEGIN
        r := '';
        FOR p IN SELECT name FROM servers WHERE id =
            ANY((select allowed_sources FROM server_allowed_sources WHERE server_id = xid)::INT[]) LOOP
            r := r || p || ',';
        END LOOP;
        RETURN rtrim(r,',');
    END;
$delim$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION in_submission_period(server_id integer) RETURNS BOOLEAN AS $delim$
    DECLARE
    t boolean;
    BEGIN
       SELECT
        to_char(current_timestamp, 'HH24')::int >= start_submission_period
        AND
        to_char(current_timestamp, 'HH24')::int <= end_submission_period INTO t
        FROM servers WHERE id = server_id;
        RETURN t;
    END;
$delim$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION xml_pretty(xml text)
RETURNS xml AS $$
        SELECT xslt_process($1,
'<xsl:stylesheet version=\1.0\ xmlns:xsl=\http://www.w3.org/1999/XSL/Transform\>
<xsl:strip-space elements=\*\ />
<xsl:output method=\xml\ indent=\yes\ />
<xsl:template match=\node() | @*\>
<xsl:copy>
<xsl:apply-templates select=\node() | @*\ />
</xsl:copy>
</xsl:template>
</xsl:stylesheet>')::xml
$$ LANGUAGE SQL IMMUTABLE STRICT;

CREATE OR REPLACE FUNCTION is_valid_json(p_json text)
  RETURNS BOOLEAN
AS $$
        BEGIN
                return (p_json::json is not null);
                EXCEPTION WHEN OTHERS THEN
                        return false;
        END;
$$ LANGUAGE plpgsql IMMUTABLE;

CREATE OR REPLACE FUNCTION pp_json(j TEXT, sort_keys BOOLEAN = TRUE, indent TEXT = '    ')
RETURNS TEXT AS $delim$
  import simplejson as json
  if not j:
      return ''
  return json.dumps(json.loads(j), sort_keys=sort_keys, indent=indent)
$delim$ LANGUAGE plpython3u;

CREATE OR REPLACE FUNCTION body_pprint(body text)
    RETURNS TEXT AS $$
    BEGIN
        IF xml_is_well_formed_document(body) THEN
            return xml_pretty(body)::text;
        ELSIF is_valid_json(body) THEN
            return pp_json(body, 't', '    ');
		 ELSE
			 return body;
        END IF;
    END;
$$ LANGUAGE plpgsql;

-- Data Follows
INSERT INTO user_roles(name, descr)
VALUES('Administrator','For the Administrators'),
('SMS User', 'For SMS third party apps');

INSERT INTO user_role_permissions(user_role, sys_module,sys_perms)
VALUES
        ((SELECT id FROM user_roles WHERE name ='Administrator'),'Users','rmad');

INSERT INTO users(firstname,lastname,username,password,email,user_role,is_system_user)
VALUES
        ('Samuel','Sekiwere','admin',crypt('admin',gen_salt('bf')),'sekiskylink@gmail.com',
        (SELECT id FROM user_roles WHERE name ='Administrator'),'t'),
        ('DHIS2','PMTCT','dhis2_pmtct',crypt('dhis2',gen_salt('bf')),'',
        (SELECT id FROM user_roles WHERE name ='SMS User'),'f');

INSERT INTO server_allowed_sources (server_id, allowed_sources)
    VALUES((SELECT id FROM servers where name = 'dhis2'),
        (SELECT array_agg(id) FROM servers WHERE name in ('mtrack', 'mtrackpro')));
`

var keyTables = []string{
	"servers", "users", "requests", "user_roles", "server_allowed_sources"}

func init() {
	log.Printf(
		"Trying to initialize database schema. check tables %v", strings.Join(keyTables, ", "))
	checkTables()
}

func checkTables() int {
	dbconn := GetDB()

	keyTableMissing := false

	for _, table := range keyTables {
		query := fmt.Sprintf("SELECT id FROM %s", table)
		_, err := dbconn.Queryx(query)
		if err != nil {
			log.Printf("Checking presence of key table: %v -> Missing", table)
			keyTableMissing = true
		} else {
			log.Printf("Checking presence of key table: %v -> Exists", table)
		}
	}
	if keyTableMissing {
		tx := db.MustBegin()
		dbconn.MustExec(schema)
		tx.Commit()

	}

	return 0
}
