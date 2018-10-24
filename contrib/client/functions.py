class Section(object):
    def __init__(self, header, values):
        self.header = header
        self.values = values
    
    def _lines(self, indent, ls):
        ls.append((indent, self.header))
        for v in self.values:
            if isinstance(v, Section):
                v._lines(indent + 1, ls)
            else:
                ls.append((indent + 1, v))
    
    def lines(self):
        ls = []
        self._lines(0, ls)
        return ls
    
    def pretty_print(self):
        return '\n'.join("\t" * indent + str(v) for (indent, v) in self.lines())

class IndentedList(object):
    def __init__(self, values):
        self.values = values
    
    def _lines(self, indent, ls):
        for (k, vs) in self.values:
            ls.append((indent, k))
            ls.extend((indent + 1, v) for v in vs)
        return ls
    
    def lines(self):
        ls = []
        self._lines(ls)
        return ls

def get_image_shas(dump):
    return dump.images.keys()

def get_pod_names(dump):
    return dump.pods.keys()

def get_container_image(cont):
    img = cont.image
    vs = []
    if img.scan_results is not None:
        sr = img.scan_results
        vs.append("Policy violations: {}".format(sr['PolicyStatus']['ComponentVersionStatusCounts']['IN_VIOLATION']))
        vs.append("High-risk vulnerabilities: {}".format(sr['RiskProfile']['Categories']['VULNERABILITY']['StatusCounts']['HIGH']))
    return Section("{}: {}".format(cont.name, cont.image.sha), vs)

def get_pod_images(pod):
    return Section(pod.name, map(get_container_image, pod.containers))

def get_all_pod_images(dump):
    sections = []
    for pod in dump.pods.values():
        sections.append(get_pod_images(pod))
    return "\n".join(s.pretty_print() for s in sections)

def namespace_pod_images(dump, namespace=None):
    if namespace is None:
        ns = dump.ns.keys()
    else:
        ns = [namespace]
    sections = []
    for nsname in ns:
        pods = dump.ns.get(nsname, [])
        sections.append(Section("namespace: " + nsname, map(get_pod_images, pods)))
    return "\n".join(s.pretty_print() for s in sections)

def namespace_images(dump, namespace=None):
    if namespace is None:
        ns = dump.ns.keys()
    else:
        ns = [namespace]
    sections = []
    for nsname in ns:
        sections.append(Section("namespace: " + nsname, dump.namespace_images(nsname)))
    return "\n".join(s.pretty_print() for s in sections)
