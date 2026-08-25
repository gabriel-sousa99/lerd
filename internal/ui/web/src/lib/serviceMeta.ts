import { derived } from 'svelte/store';
import { presets } from '$stores/presets';
import { services } from '$stores/services';

export interface ServiceMeta {
  category?: string;
  icon?: string;
  color?: string;
  // The preset a service was installed from ("mariadb" for "mariadb-11-8"),
  // which is what the store icon is keyed by.
  preset?: string;
  // The services an admin UI fronts, which is what it borrows a mark from when
  // it ships none of its own.
  admin_for?: string[];
}

function metaOf(s: {
  category?: string;
  icon?: string;
  color?: string;
  preset?: string;
  admin_for?: string[];
}): ServiceMeta {
  return {
    category: s.category,
    icon: s.icon,
    color: s.color,
    preset: s.preset,
    admin_for: s.admin_for
  };
}

// Name to declared metadata, so a component holding only a service name (a
// site's service chip) still draws the right icon. Installed services win over
// presets: a versioned member like "mariadb-11-8" exists only in the service
// list, already resolved through its preset server-side.
export const serviceMeta = derived([presets, services], ([$presets, $services]) => {
  const meta = new Map<string, ServiceMeta>();
  for (const p of $presets) {
    if (p.category || p.icon || p.color || p.admin_for)
      meta.set(p.name, { ...metaOf(p), preset: p.name });
  }
  for (const s of $services) {
    if (s.category || s.icon || s.color || s.preset || s.admin_for) meta.set(s.name, metaOf(s));
  }
  return meta;
});
