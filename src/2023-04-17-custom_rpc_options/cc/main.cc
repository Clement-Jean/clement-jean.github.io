#include <iostream>
#include <optional>
#include <functional>

#include "proto/hello.pb.h"
#include "proto/world.pb.h"

#include <google/protobuf/service.h>

using namespace google::protobuf;

template<typename OPT_T>
std::optional<OPT_T> get_option_value(
	const MethodOptions &opts,
	const auto &id
) {
	return opts.HasExtension(id) ?
		std::optional(opts.GetExtension(id)) :
		std::nullopt;
}

std::optional<const MethodDescriptor *> get_service_method(
	const ServiceDescriptor *sd,
	const std::function<bool(const MethodDescriptor *)> &predicate
) {
	if (!sd)
		return std::nullopt;

	for (int i = 0; i < sd->method_count(); ++i) {
		auto md = sd->method(i);

		if (predicate(md))
			return md;
	}

	return std::nullopt;
}

template<typename U>
std::optional<U> get_method_option_value(
	const ServiceDescriptor *sd,
	const auto &id
) {
	if (!sd)
		return std::nullopt;

	std::optional<U> value;

	get_service_method(sd, [&value, &id](const MethodDescriptor *md) -> bool {
		auto opts = md->options();

		if (auto tmp = get_option_value<U>(opts, id))
			value = tmp;

		return value != std::nullopt;
	});

	return value;
}

int main() {
	auto sd = HelloWorldService::descriptor();
	auto world = get_method_option_value<std::string>(sd, hello);

	if (world)
		std::cout << *world << std::endl;
	return 0;
}