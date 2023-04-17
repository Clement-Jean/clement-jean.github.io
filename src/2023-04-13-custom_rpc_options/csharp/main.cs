#nullable enable annotations

using System;
using System.Linq;
using System.Collections.Generic;
using pb = global::Google.Protobuf;

static class HelloWorldApp {

	static private T GetOptionValue<T>(
		this pb.Reflection.MethodDescriptor md,
		pb::Extension<pb.Reflection.MethodOptions, T> id
	) => md.GetOptions().GetExtension(id);

	static private IEnumerable<pb.Reflection.MethodDescriptor> GetServiceMethod(
		this IEnumerable<pb.Reflection.ServiceDescriptor> services,
		Func<pb.Reflection.MethodDescriptor, bool> predicate
	) => from svc in services
			 from method in svc.Methods
			 where predicate(method)
			 select method;

	static private T GetMethodOptionValue<T>(
		this pb.Reflection.FileDescriptor fd,
		pb::Extension<pb.Reflection.MethodOptions, T> id
	) => fd.Services
				 .GetServiceMethod(md => md.GetOptionValue(id) != null)
				 .FirstOrDefault()
				 .GetOptionValue(id);

	static public void Main(String[] args)
	{
		var id = HelloExtensions.Hello;
		string world = WorldReflection.Descriptor.GetMethodOptionValue(id);

		if (world.Length != 0)
			System.Console.WriteLine(world);
	}
}