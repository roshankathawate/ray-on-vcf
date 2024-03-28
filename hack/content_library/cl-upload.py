import requests
import os
import uuid
from urllib.request import Request, urlopen
import ssl

from com.vmware.cis_client import Session
from vmware.vapi.lib.connect import get_requests_connector
from vmware.vapi.security.session import create_session_security_context
from vmware.vapi.security.user_password import create_user_password_security_context
from com.vmware.content.library.item.updatesession_client import (
    File as UpdateSessionFile,
)

from vmware.vapi.stdlib.client.factories import StubConfigurationFactory

from com.vmware.content_client import LibraryModel, LocalLibrary
from com.vmware.content.library_client import (
    StorageBacking,
    ItemModel,
    Item,
    PublishInfo,
)
from com.vmware.content.library.item_client import UpdateSession, UpdateSessionModel
from com.vmware.vcenter_client import Datastore


def create_unverified_session(session, suppress_warning=True):
    session.verify = False
    if suppress_warning:
        # Suppress unverified https request warnings
        from requests.packages.urllib3.exceptions import InsecureRequestWarning

        requests.packages.urllib3.disable_warnings(InsecureRequestWarning)
    return session


def createStubConfig(host, user, pwd):
    host_url = "https://{}/api".format(host)

    session = requests.Session()
    session = create_unverified_session(session, True)

    connector = get_requests_connector(session=session, url=host_url)
    user_password_security_context = create_user_password_security_context(user, pwd)

    stub_config = StubConfigurationFactory.new_std_configuration(connector)
    stub_config.connector.set_security_context(user_password_security_context)

    # Create the stub for the session service and login by creating a session.
    session_svc = Session(stub_config)
    session_id = session_svc.create()

    # Successful authentication.  Store the session identifier in the security
    # context of the stub and use that for all subsequent remote requests
    session_security_context = create_session_security_context(session_id)
    stub_config.connector.set_security_context(session_security_context)

    return stub_config


def generate_random_uuid():
    return str(uuid.uuid4())


def get_files_map(ovf_location):
    ovf_files_map = {}
    ovf_dir = os.path.abspath(
        os.path.join(os.path.dirname(os.path.realpath(__file__)), ovf_location)
    )
    for file_name in os.listdir(ovf_dir):
        if (
            file_name.endswith(".ovf")
            or file_name.endswith(".vmdk")
            or file_name.endswith(".nvram")
            or file_name.endswith(".iso")
        ):
            ovf_files_map[file_name] = os.path.join(ovf_dir, file_name)
    return ovf_files_map


def create_local_library(stub_config, datastore, lib_name):

    # fetch datastore id using name.

    filter_spec = Datastore.FilterSpec(names=set([datastore]))
    dsobj = Datastore(stub_config)

    datastores = dsobj.list(filter_spec)

    assert datastores is not None
    assert len(datastores) is not None

    datastore_id = datastores[0].datastore
    print("Datastore found with following ID: {0}".format(datastore_id))

    # Build the storage backing for the library to be created
    storage_backings = []
    storage_backing = StorageBacking(
        type=StorageBacking.Type.DATASTORE, datastore_id=datastore_id
    )
    storage_backings.append(storage_backing)

    # Build the specification for the library to be created, which is also a publisher.
    create_spec = LibraryModel()
    create_spec.name = lib_name
    create_spec.description = "Local library backed by VC datastore"
    create_spec.type = create_spec.LibraryType.LOCAL
    create_spec.storage_backings = storage_backings
    create_spec.publish_info = PublishInfo(
        authentication_method=PublishInfo.AuthenticationMethod.NONE, published=True
    )

    # Returns the service for managing local libraries
    local_library_service = LocalLibrary(stub_config)

    # Create a local content library backed the VC datastore using vAPIs
    library_id = local_library_service.create(
        create_spec=create_spec, client_token=generate_random_uuid()
    )
    print("Local library created: ID: {0}".format(library_id))

    return library_id


def create_local_library_ovf_item(stub_config, library_id, lib_item_name, description):

    local_library_service = LocalLibrary(stub_config)

    local_library = local_library_service.get(library_id)

    # create item model.
    lib_item_spec = ItemModel()
    lib_item_spec.name = lib_item_name
    lib_item_spec.description = description
    lib_item_spec.library_id = local_library.id
    lib_item_spec.type = "ovf"

    # Create a new library item in the content library for uploading the files
    library_item_service = Item(stub_config)

    library_item_id = library_item_service.create(
        create_spec=lib_item_spec, client_token=generate_random_uuid()
    )

    assert library_item_id is not None
    print("Library item created id: {0}".format(library_item_id))

    return library_item_id


def upload_files_in_session(stub_config, files_map, session_id):

    upload_file_service = UpdateSessionFile(stub_config)

    for f_name, f_path in files_map.items():
        file_spec = upload_file_service.AddSpec(
            name=f_name,
            source_type=UpdateSessionFile.SourceType.PUSH,
            size=os.path.getsize(f_path),
        )
        file_info = upload_file_service.add(session_id, file_spec)
        # Upload the file content to the file upload URL
        with open(f_path, "rb") as local_file:
            request = Request(file_info.upload_endpoint.uri, local_file)
            request.add_header("Cache-Control", "no-cache")
            request.add_header("Content-Length", "{0}".format(os.path.getsize(f_path)))
            request.add_header("Content-Type", "text/ovf")

            # Python 2.7.9 has stronger SSL certificate validation,
            # so we need to pass in a context when dealing with
            # self-signed certificates.
            context = ssl._create_unverified_context()
            urlopen(request, context=context)


def upload_files_to_item(stub_config, ovf_location, library_item_id):

    ovf_files_map = get_files_map(ovf_location)

    upload_service = UpdateSession(stub_config)
    session_id = upload_service.create(
        create_spec=UpdateSessionModel(library_item_id=library_item_id),
        client_token=generate_random_uuid(),
    )

    upload_files_in_session(stub_config, ovf_files_map, session_id)
    upload_service.complete(session_id)
    upload_service.delete(session_id)


# ========================

import argparse
from getpass import getpass

parser = argparse.ArgumentParser(
    description="Upload vmray cluster operator docker image to resgistry"
)
parser.add_argument("-i", dest="vc_host", type=str, required=True)
parser.add_argument("-u", dest="username", type=str, required=True)
parser.add_argument("-ds", dest="datastore", type=str, required=True)
parser.add_argument("-ln", dest="library_name", type=str, required=True)
parser.add_argument("-lin", dest="library_item_name", type=str, required=True)
parser.add_argument("-lid", dest="library_item_description", type=str, required=True)
parser.add_argument("-p", dest="ovf_file_path", type=str, required=True)
args = parser.parse_args()

# Example : python cl-upload.py -i 'x.x.x.x' -u 'administrator@vsphere.local' -ds 'datastore-name' -ln 'ray-svamshik' -lin 'ray-frozen-debian' -lid 'item to hold ray debian ovf' -p /Users/svamshik/Downloads/ray-frozen-debian/

stub_config = createStubConfig(args.vc_host, args.username, getpass())

lib_id = create_local_library(stub_config, args.datastore, args.library_name)
lib_item_id = create_local_library_ovf_item(
    stub_config, lib_id, args.library_item_name, args.library_item_description
)

upload_files_to_item(stub_config, args.ovf_file_path, lib_item_id)
