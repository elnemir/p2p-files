"use client";
import { useBranding } from "./BrandingProvider";
export default function Brand() {
  const { value: b } = useBranding();
  const light = b.logoLight || b.logoDark;
  const dark = b.logoDark || b.logoLight;
  return (
    <a className="brand" href="/" aria-label={b.productName}>
      {light ? (
        <>
          <img
            className="brand-logo brand-logo-light"
            src={light}
            alt={b.productName}
          />
          <img
            className="brand-logo brand-logo-dark"
            src={dark}
            alt={b.productName}
          />
        </>
      ) : (
        <span className="brand-name">{b.productName}</span>
      )}
    </a>
  );
}
