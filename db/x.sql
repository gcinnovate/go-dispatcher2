CREATE TABLE servers(
    id serial PRIMARY KEY NOT NULL,
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

CREATE TABLE requests(
    id bigserial PRIMARY KEY NOT NULL,
    source INTEGER REFERENCES servers(id), -- source app/server
    destination INTEGER REFERENCES servers(id), -- source app/server
    body TEXT NOT NULL DEFAULT '',
    body_is_query_param BOOLEAN NOT NULL DEFAULT 'f',
    url_suffix TEXT DEFAULT '', -- if present, it is added to API url 
    ctype TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'ready' CHECK( status IN('pending', 'ready', 'inprogress', 'failed', 'error', 'expired', 'completed', 'canceled')),

    statuscode text DEFAULT '',
    retries INTEGER NOT NULL DEFAULT 0,
    errors TEXT DEFAULT '', -- indicative response message
    submissionid INTEGER NOT NULL DEFAULT 0, -- message_id in source app -> helpful when check for already sent submissions
    week TEXT DEFAULT '', -- reporting week
    month TEXT DEFAULT '', -- reporting month
    year INTEGER, -- year of submission
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

INSERT INTO servers (name, username, password, ipaddress, url, auth_method)
    VALUES
        ('localhost', 'tester', 'foobar', '127.0.0.1', 'http://localhost:8080/test', 'Basic Auth'),
        ('mtrack', 'tester', 'foobar', '127.0.0.1', 'http://localhost:8080/test', 'Basic Auth'),
        ('mtrackpro', 'tester', 'foobar', '127.0.0.1', 'http://localhost:8080/test', 'Basic Auth'),
        ('dhis2', 'tester', 'foobar', '127.0.0.1', 'http://localhost/api/dataValueSets', 'Basic Auth');

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

