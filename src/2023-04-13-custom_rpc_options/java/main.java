package com.clementjean.test;

import com.google.protobuf.Descriptors;
import com.google.protobuf.DescriptorProtos;
import com.google.protobuf.GeneratedMessage;
import java.util.Optional;
import java.util.function.Function;
import java.util.concurrent.atomic.AtomicReference;

class HelloWorldApp {
    private static <T> Optional<T> getOptionValue(
        DescriptorProtos.MethodOptions opts,
        GeneratedMessage.GeneratedExtension<DescriptorProtos.MethodOptions, ?> id
    ) {
        return opts.hasExtension(id) ?
              Optional.of((T)opts.getExtension(id)) :
              Optional.empty();
    }

    private static <T> Optional<Descriptors.MethodDescriptor> getServiceMethod(
        Descriptors.ServiceDescriptor sd,
        Function<Descriptors.MethodDescriptor, Boolean> predicate
    ) {
        for (int i = 0; i < sd.getMethods().size(); ++i) {
            Descriptors.MethodDescriptor method = sd.getMethods().get(i);

            if (predicate.apply(method))
                return Optional.of(method);
        }

        return Optional.empty();
    }

    private static <T> Optional<T> getMethodOptionValue(
        Descriptors.ServiceDescriptor sd,
        GeneratedMessage.GeneratedExtension<DescriptorProtos.MethodOptions, ?> id
    ) {
        AtomicReference<Optional<T>> value = new AtomicReference<>(Optional.empty());

        getServiceMethod(sd, md -> {
            DescriptorProtos.MethodOptions opts = md.getOptions();
            Optional<T> tmp = getOptionValue(opts, id);

            if (tmp.isPresent())
                value.set(tmp);

            return value.get().isPresent();
        });

		return value.get();
	}

    private static <T> Optional<T> getMethodExtension(
        Descriptors.FileDescriptor fd,
        GeneratedMessage.GeneratedExtension<DescriptorProtos.MethodOptions, ?> id
    ) {
        for (int i = 0; i < fd.getServices().size(); ++i) {
            Descriptors.ServiceDescriptor sd = fd.getServices().get(i);
            Optional<T> world = getMethodOptionValue(sd, Hello.hello);

            if (world.isPresent())
                return world;
        }

        return Optional.empty();
    }

    public static void main(String[] args) {
        Descriptors.FileDescriptor fd = World.getDescriptor();
        Optional<String> world = getMethodExtension(fd, Hello.hello);

        world.ifPresent(w -> System.out.println(w));
    }
}