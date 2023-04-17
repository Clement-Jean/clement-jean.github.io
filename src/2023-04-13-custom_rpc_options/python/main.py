from proto.world_pb2 import DESCRIPTOR

def get_option_value(opts, id):
  for field in opts.ListFields():
    (desc, value) = field

    if value != "" and desc.name == id:
      return value

def get_service_method(sd, predicate):
  for method_name in sd.methods_by_name:
    md = sd.methods_by_name[method_name]

    if predicate(md):
      return md

def get_method_option_value(sd, id):
  md = get_service_method(sd, lambda md: get_option_value(md.GetOptions(), id) != None)

  return get_option_value(md.GetOptions(), id)

def get_method_extension(fd, id):
  for svc_name in fd.services_by_name:
    sd = fd.services_by_name[svc_name]
    value = get_method_option_value(sd, id)

    if value != None:
      return value

print(get_method_extension(DESCRIPTOR, "hello"))