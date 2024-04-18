# Copyright (c) 2024 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

import requests
from enum import Enum

from com.vmware.cis_client import Session

from vmware.vapi.security.session import create_session_security_context
from vmware.vapi.lib.connect import get_requests_connector
from vmware.vapi.stdlib.client.factories import StubConfigurationFactory
from vmware.vapi.security.client.security_context_filter import \
    LegacySecurityContextFilter
from vmware.vapi.security.user_password import \
    create_user_password_security_context


"""
Creates a session and returns the stub configuration
"""


def get_configuration(server, username, password, skipVerification):
    session = get_unverified_session() if skipVerification else None
    if not session:
        session = requests.Session()
    host_url = "https://{}/api".format(server)
    sec_ctx = create_user_password_security_context(username,
                                                    password)
    session_svc = Session(
        StubConfigurationFactory.new_std_configuration(
            get_requests_connector(
                session=session, url=host_url,
                provider_filter_chain=[
                    LegacySecurityContextFilter(
                        security_context=sec_ctx)])))
    session_id = session_svc.create()
    print("Session ID : ", session_id)
    sec_ctx = create_session_security_context(session_id)
    stub_config = StubConfigurationFactory.new_std_configuration(
        get_requests_connector(
            session=session, url=host_url,
            provider_filter_chain=[
                LegacySecurityContextFilter(
                    security_context=sec_ctx)]))
    return stub_config

def get_unverified_session():
    """
    vCenter provisioned internally have SSH certificates
    expired so we use unverified session. Find out what
    could be done for production.

    Get a requests session with cert verification disabled.
    Also disable the insecure warnings message.
    Note this is not recommended in production code.
    @return: a requests session with verification disabled.
    """
    session = requests.session()
    session.verify = False
    requests.packages.urllib3.disable_warnings()
    return session

class Constants:
    class SessionType(Enum):
        VERIFIED = "verified"
        UNVERIFIED = "unverified"
